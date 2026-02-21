// SPDX-License-Identifier: MIT
//go:build ignore
// +build ignore

package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/scanner"
	"go/token"
	"io/fs"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

type distribution struct {
	Count int     `json:"count"`
	Min   float64 `json:"min"`
	Mean  float64 `json:"mean"`
	P95   float64 `json:"p95"`
	Max   float64 `json:"max"`
}

type lineMetrics struct {
	LOC            int     `json:"loc"`
	SLOC           int     `json:"sloc"`
	LLOC           int     `json:"lloc"`
	CommentLines   int     `json:"comment_lines"`
	CommentDensity float64 `json:"comment_density_percent"`
}

type halsteadMetrics struct {
	Vocabulary int     `json:"halstead_vocabulary"`
	Length     int     `json:"halstead_length"`
	Volume     float64 `json:"halstead_volume"`
	Difficulty float64 `json:"halstead_difficulty"`
	Effort     float64 `json:"halstead_effort"`
	Operators  int     `json:"operators_total"`
	Operands   int     `json:"operands_total"`
}

type ooAggregate struct {
	Classes int          `json:"classes"`
	WMC     distribution `json:"wmc"`
	DIT     distribution `json:"dit"`
	NOC     distribution `json:"noc"`
	CBO     distribution `json:"cbo"`
	LCOM    distribution `json:"lcom"`
	RFC     distribution `json:"rfc"`
}

type couplingMetrics struct {
	AfferentCoupling distribution `json:"afferent_coupling"`
	EfferentCoupling distribution `json:"efferent_coupling"`
	Instability      distribution `json:"instability"`
}

type fanMetrics struct {
	FanIn  distribution `json:"fan_in"`
	FanOut distribution `json:"fan_out"`
}

type docsMetrics struct {
	ExportedSymbols           int     `json:"exported_symbols"`
	DocumentedExportedSymbols int     `json:"documented_exported_symbols"`
	DocumentationCoverage     float64 `json:"documentation_coverage_percent"`
	DocumentationThreshold    float64 `json:"documentation_threshold_percent"`
	MeetsThreshold            bool    `json:"meets_threshold"`
}

type duplicationMetrics struct {
	WindowSize         int     `json:"window_size_lines"`
	DuplicateWindows   int     `json:"duplicate_windows"`
	ApproxDuplicateLOC int     `json:"approx_duplicate_loc"`
	DuplicationPercent float64 `json:"duplication_percent"`
}

type coverageMetrics struct {
	Available         bool    `json:"available"`
	TotalStatements   int     `json:"total_statements"`
	CoveredStatements int     `json:"covered_statements"`
	CodeCoverage      float64 `json:"code_coverage_percent"`
}

type debtMetrics struct {
	TodoCount               int     `json:"todo_count"`
	HighCyclomaticFunctions int     `json:"high_cyclomatic_functions"`
	HighCognitiveFunctions  int     `json:"high_cognitive_functions"`
	DebtHoursProxy          float64 `json:"technical_debt_hours_proxy"`
}

type changeMetrics struct {
	Available             bool    `json:"available"`
	Window                string  `json:"window"`
	ChangedFiles          int     `json:"changed_files"`
	TotalFiles            int     `json:"total_files"`
	ChangeInstability     float64 `json:"change_instability_percent"`
	MostChangedPackage    string  `json:"most_changed_package"`
	MostChangedPackagePct float64 `json:"most_changed_package_percent"`
}

type complexityMetrics struct {
	Cyclomatic  distribution `json:"cyclomatic_complexity"`
	Cognitive   distribution `json:"cognitive_complexity"`
	Essential   distribution `json:"essential_complexity_proxy"`
	Pathologic  distribution `json:"pathological_complexity_proxy"`
	Nesting     distribution `json:"nesting_depth"`
	MaintainIdx float64      `json:"maintainability_index"`
}

type report struct {
	GeneratedAt string             `json:"generated_at"`
	Root        string             `json:"root"`
	Module      string             `json:"module"`
	Files       int                `json:"files_analyzed"`
	Lines       lineMetrics        `json:"lines"`
	Complexity  complexityMetrics  `json:"complexity"`
	Halstead    halsteadMetrics    `json:"halstead"`
	OO          ooAggregate        `json:"object_oriented_metrics"`
	Coupling    couplingMetrics    `json:"coupling"`
	Fan         fanMetrics         `json:"fan"`
	Docs        docsMetrics        `json:"documentation"`
	Duplication duplicationMetrics `json:"duplication"`
	Coverage    coverageMetrics    `json:"coverage"`
	Debt        debtMetrics        `json:"technical_debt"`
	Change      changeMetrics      `json:"change"`
	Hotspots    []hotspot          `json:"hotspots"`
	Notes       []string           `json:"notes"`
}

type hotspot struct {
	ID         string  `json:"id"`
	Package    string  `json:"package"`
	Name       string  `json:"name"`
	Cyclomatic int     `json:"cyclomatic"`
	Cognitive  int     `json:"cognitive"`
	Nesting    int     `json:"nesting"`
	Essential  int     `json:"essential"`
	Pathologic float64 `json:"pathological_proxy"`
}

type classMetric struct {
	name      string
	fields    map[string]struct{}
	embedded  []string
	methods   map[string]*methodMetric
	cboSet    map[string]struct{}
	doc       bool
	exported  bool
	pkg       string
	receiver  string
	recvVar   string
	callSet   map[string]struct{}
	fieldSet  map[string]struct{}
	complex   int
	cognitive int
}

type methodMetric struct {
	name      string
	complex   int
	cognitive int
	calls     map[string]struct{}
	fields    map[string]struct{}
}

type functionMetric struct {
	id        string
	pkg       string
	name      string
	complex   int
	cognitive int
	nesting   int
	essential int
	patho     float64
	calls     map[string]struct{}
}

func main() {
	rootFlag := flag.String("root", ".", "project root")
	includeTests := flag.Bool("include-tests", false, "include *_test.go files")
	coverageFile := flag.String("coverage-file", "", "optional go coverprofile path")
	docThreshold := flag.Float64("doc-threshold", 80.0, "documentation coverage threshold percent")
	dupWindow := flag.Int("dup-window", 6, "duplication sliding window size")
	changeWindow := flag.String("change-window", "90.days", "git log window for change instability")
	topHotspots := flag.Int("top-hotspots", 25, "number of complexity hotspots to include in report")
	flag.Parse()

	root, err := filepath.Abs(*rootFlag)
	if err != nil {
		fail(err)
	}

	modulePath, _ := readModulePath(filepath.Join(root, "go.mod"))
	files, err := discoverGoFiles(root, *includeTests)
	if err != nil {
		fail(err)
	}

	if len(files) == 0 {
		fail(errors.New("no go files found"))
	}

	fset := token.NewFileSet()

	line := lineMetrics{}
	todoCount := 0
	operators := map[string]int{}
	operands := map[string]int{}

	functions := make(map[string]*functionMetric)
	functionNameIndex := make(map[string][]string)
	classes := make(map[string]*classMetric)
	pkgImports := make(map[string]map[string]struct{})

	exportedSymbols := 0
	documentedExportedSymbols := 0

	dupByFile := make(map[string][]string)
	fileMIVals := make([]float64, 0, len(files))

	for _, path := range files {
		relDir := filepath.Dir(strings.TrimPrefix(path, root+string(filepath.Separator)))
		if relDir == "." {
			relDir = ""
		}
		pkgImportPath := packageImportPath(modulePath, relDir)
		src, err := os.ReadFile(path)
		if err != nil {
			fail(err)
		}

		loc, sloc, commentLines, todos, normalized := analyzeLines(src)
		line.LOC += loc
		line.SLOC += sloc
		line.CommentLines += commentLines
		todoCount += todos
		dupByFile[path] = normalized

		analyzeHalstead(path, src, operators, operands)

		fileNode, err := parser.ParseFile(fset, path, src, parser.ParseComments)
		if err != nil {
			continue
		}
		fileComplexityTotal := 0
		fileOps := make(map[string]int)
		fileOperands := make(map[string]int)
		analyzeHalstead(path, src, fileOps, fileOperands)

		if _, ok := pkgImports[pkgImportPath]; !ok {
			pkgImports[pkgImportPath] = make(map[string]struct{})
		}
		for _, imp := range fileNode.Imports {
			impPath, _ := strconv.Unquote(imp.Path.Value)
			if impPath != "" {
				pkgImports[pkgImportPath][impPath] = struct{}{}
			}
		}

		exported, documented := countExportedSymbols(fileNode)
		exportedSymbols += exported
		documentedExportedSymbols += documented

		ast.Inspect(fileNode, func(n ast.Node) bool {
			switch d := n.(type) {
			case *ast.FuncDecl:
				if d.Body == nil {
					return true
				}

				fnID, fnName, recvType, recvVar := functionIdentity(pkgImportPath, d)
				complex, cognitive, nesting, calls, fields := analyzeFunctionBody(d.Body, recvVar)
				line.LLOC += countStatements(d.Body)

				essential := complex - maxInt(0, nesting-1)
				if essential < 1 {
					essential = 1
				}
				patho := 0.0
				if complex > 0 {
					patho = float64(cognitive) / float64(complex)
				}

				functions[fnID] = &functionMetric{
					id:        fnID,
					pkg:       pkgImportPath,
					name:      fnName,
					complex:   complex,
					cognitive: cognitive,
					nesting:   nesting,
					essential: essential,
					patho:     patho,
					calls:     calls,
				}
				functionNameIndex[fnName] = append(functionNameIndex[fnName], fnID)
				fileComplexityTotal += complex

				if recvType != "" {
					classKey := pkgImportPath + "." + recvType
					cm := ensureClass(classes, classKey, pkgImportPath, recvType)
					cm.methods[d.Name.Name] = &methodMetric{
						name:      d.Name.Name,
						complex:   complex,
						cognitive: cognitive,
						calls:     calls,
						fields:    fields,
					}
					for c := range calls {
						cm.callSet[c] = struct{}{}
						if strings.Contains(c, ".") {
							cm.cboSet[c] = struct{}{}
						}
					}
					for f := range fields {
						cm.fieldSet[f] = struct{}{}
					}
				}
			}
			return true
		})

		for _, decl := range fileNode.Decls {
			gen, ok := decl.(*ast.GenDecl)
			if !ok || gen.Tok != token.TYPE {
				continue
			}
			for _, spec := range gen.Specs {
				ts, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}
				st, ok := ts.Type.(*ast.StructType)
				if !ok {
					continue
				}
				classKey := pkgImportPath + "." + ts.Name.Name
				cm := ensureClass(classes, classKey, pkgImportPath, ts.Name.Name)
				cm.exported = ast.IsExported(ts.Name.Name)
				cm.doc = hasDoc(gen.Doc) || hasDoc(ts.Doc)

				for _, f := range st.Fields.List {
					if len(f.Names) == 0 {
						embName := exprName(f.Type)
						if embName != "" {
							cm.embedded = append(cm.embedded, embName)
						}
						continue
					}
					for _, n := range f.Names {
						cm.fields[n.Name] = struct{}{}
					}
				}
			}
		}
		fileHal := buildHalstead(fileOps, fileOperands)
		fileMIVals = append(fileMIVals, maintainabilityIndex(fileHal.Volume, float64(fileComplexityTotal), float64(maxInt(1, sloc))))
	}

	line.CommentDensity = percent(float64(line.CommentLines), float64(maxInt(1, line.LOC)))

	complexVals := make([]float64, 0, len(functions))
	cognitiveVals := make([]float64, 0, len(functions))
	nestingVals := make([]float64, 0, len(functions))
	essentialVals := make([]float64, 0, len(functions))
	pathoVals := make([]float64, 0, len(functions))
	highCyclo := 0
	highCog := 0

	fanOutVals := make([]float64, 0, len(functions))
	fanIn := make(map[string]int)

	for _, fn := range functions {
		complexVals = append(complexVals, float64(fn.complex))
		cognitiveVals = append(cognitiveVals, float64(fn.cognitive))
		nestingVals = append(nestingVals, float64(fn.nesting))
		essentialVals = append(essentialVals, float64(fn.essential))
		pathoVals = append(pathoVals, fn.patho)
		fanOutVals = append(fanOutVals, float64(len(fn.calls)))
		if fn.complex > 15 {
			highCyclo++
		}
		if fn.cognitive > 20 {
			highCog++
		}

		for call := range fn.calls {
			shortCall := shortCallName(call)
			if targets, ok := functionNameIndex[shortCall]; ok {
				if len(targets) == 1 {
					fanIn[targets[0]]++
					continue
				}
				for _, target := range targets {
					if strings.HasPrefix(target, fn.pkg+".") {
						fanIn[target]++
					}
				}
			}
		}
	}

	fanInVals := make([]float64, 0, len(functions))
	for id := range functions {
		fanInVals = append(fanInVals, float64(fanIn[id]))
	}

	hal := buildHalstead(operators, operands)
	mi := calcDistribution(fileMIVals).Mean

	wmcVals, ditVals, nocVals, cboVals, lcomVals, rfcVals := computeOOMetrics(classes, pkgImports)

	caVals, ceVals, instVals := computeCoupling(pkgImports, modulePath)

	dup := computeDuplication(dupByFile, *dupWindow, line.SLOC)

	coverage := parseCoverage(*coverageFile)

	debt := debtMetrics{
		TodoCount:               todoCount,
		HighCyclomaticFunctions: highCyclo,
		HighCognitiveFunctions:  highCog,
		DebtHoursProxy:          float64(todoCount)*0.25 + float64(highCyclo)*0.5 + float64(highCog)*0.75 + dup.DuplicationPercent*0.1,
	}

	change := computeChangeMetrics(root, files, *changeWindow)

	docs := docsMetrics{
		ExportedSymbols:           exportedSymbols,
		DocumentedExportedSymbols: documentedExportedSymbols,
		DocumentationCoverage:     percent(float64(documentedExportedSymbols), float64(maxInt(1, exportedSymbols))),
		DocumentationThreshold:    *docThreshold,
	}
	docs.MeetsThreshold = docs.DocumentationCoverage >= docs.DocumentationThreshold

	rep := report{
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Root:        root,
		Module:      modulePath,
		Files:       len(files),
		Lines:       line,
		Complexity: complexityMetrics{
			Cyclomatic:  calcDistribution(complexVals),
			Cognitive:   calcDistribution(cognitiveVals),
			Essential:   calcDistribution(essentialVals),
			Pathologic:  calcDistribution(pathoVals),
			Nesting:     calcDistribution(nestingVals),
			MaintainIdx: mi,
		},
		Halstead: hal,
		OO: ooAggregate{
			Classes: len(classes),
			WMC:     calcDistribution(wmcVals),
			DIT:     calcDistribution(ditVals),
			NOC:     calcDistribution(nocVals),
			CBO:     calcDistribution(cboVals),
			LCOM:    calcDistribution(lcomVals),
			RFC:     calcDistribution(rfcVals),
		},
		Coupling: couplingMetrics{
			AfferentCoupling: calcDistribution(caVals),
			EfferentCoupling: calcDistribution(ceVals),
			Instability:      calcDistribution(instVals),
		},
		Fan: fanMetrics{
			FanIn:  calcDistribution(fanInVals),
			FanOut: calcDistribution(fanOutVals),
		},
		Docs:        docs,
		Duplication: dup,
		Coverage:    coverage,
		Debt:        debt,
		Change:      change,
		Hotspots:    buildHotspots(functions, *topHotspots),
		Notes: []string{
			"Essential complexity and pathological complexity are proxy metrics derived from control-flow/nesting heuristics.",
			"DIT/NOC are approximated from Go struct embedding relationships (Go has no classical inheritance).",
			"CBO/LCOM/RFC are approximations based on AST references and method usage patterns.",
			"Fan-in/fan-out use static call-site matching and may undercount dynamic dispatch and reflection.",
			"Duplication percentage is estimated via normalized sliding windows of SLOC lines.",
			"Maintainability Index uses the standard Halstead/Cyclomatic/SLOC formula.",
		},
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(rep); err != nil {
		fail(err)
	}
}

func shortCallName(call string) string {
	idx := strings.LastIndex(call, ".")
	if idx == -1 || idx == len(call)-1 {
		return call
	}
	return call[idx+1:]
}

func discoverGoFiles(root string, includeTests bool) ([]string, error) {
	var files []string
	skip := map[string]struct{}{
		".git":        {},
		".cache":      {},
		"vendor":      {},
		"tools":       {},
		"scripts":     {},
		"cortex_db":   {},
		"cortex_logs": {},
	}
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if _, ok := skip[d.Name()]; ok {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		if !includeTests && strings.HasSuffix(path, "_test.go") {
			return nil
		}
		files = append(files, path)
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(files)
	return files, nil
}

func analyzeLines(src []byte) (loc, sloc, comments, todos int, normalizedSLOC []string) {
	s := bufio.NewScanner(strings.NewReader(string(src)))
	inBlockComment := false
	for s.Scan() {
		line := s.Text()
		loc++
		trim := strings.TrimSpace(line)
		upper := strings.ToUpper(trim)
		if strings.Contains(upper, "TODO") || strings.Contains(upper, "FIXME") {
			todos++
		}

		if inBlockComment {
			comments++
			if strings.Contains(trim, "*/") {
				inBlockComment = false
			}
			continue
		}
		if trim == "" {
			continue
		}
		if strings.HasPrefix(trim, "//") {
			comments++
			continue
		}
		if strings.HasPrefix(trim, "/*") {
			comments++
			if !strings.Contains(trim, "*/") {
				inBlockComment = true
			}
			continue
		}

		sloc++
		normalized := strings.Join(strings.Fields(trim), " ")
		normalizedSLOC = append(normalizedSLOC, normalized)
	}
	return
}

func analyzeHalstead(filename string, src []byte, operators, operands map[string]int) {
	var s scanner.Scanner
	fset := token.NewFileSet()
	file := fset.AddFile(filename, -1, len(src))
	s.Init(file, src, nil, scanner.ScanComments)

	for {
		_, tok, lit := s.Scan()
		if tok == token.EOF {
			break
		}
		if tok == token.COMMENT || tok == token.SEMICOLON {
			continue
		}

		if tok == token.IDENT || tok.IsLiteral() {
			if lit == "" {
				lit = tok.String()
			}
			operands[lit]++
			continue
		}

		if tok.IsKeyword() || tok.IsOperator() {
			operators[tok.String()]++
		}
	}
}

func buildHalstead(operators, operands map[string]int) halsteadMetrics {
	n1, n2 := len(operators), len(operands)
	N1, N2 := 0, 0
	for _, c := range operators {
		N1 += c
	}
	for _, c := range operands {
		N2 += c
	}
	vocabulary := n1 + n2
	length := N1 + N2
	volume := 0.0
	if vocabulary > 0 {
		volume = float64(length) * math.Log2(float64(vocabulary))
	}
	difficulty := 0.0
	if n2 > 0 {
		difficulty = (float64(n1) / 2.0) * (float64(N2) / float64(n2))
	}
	effort := difficulty * volume

	return halsteadMetrics{
		Vocabulary: vocabulary,
		Length:     length,
		Volume:     round2(volume),
		Difficulty: round2(difficulty),
		Effort:     round2(effort),
		Operators:  N1,
		Operands:   N2,
	}
}

func countExportedSymbols(file *ast.File) (exported, documented int) {
	for _, decl := range file.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			if d.Name != nil && ast.IsExported(d.Name.Name) {
				exported++
				if hasDoc(d.Doc) {
					documented++
				}
			}
		case *ast.GenDecl:
			for _, spec := range d.Specs {
				switch s := spec.(type) {
				case *ast.TypeSpec:
					if ast.IsExported(s.Name.Name) {
						exported++
						if hasDoc(s.Doc) || hasDoc(d.Doc) {
							documented++
						}
					}
				case *ast.ValueSpec:
					for _, n := range s.Names {
						if ast.IsExported(n.Name) {
							exported++
							if hasDoc(s.Doc) || hasDoc(d.Doc) {
								documented++
							}
						}
					}
				}
			}
		}
	}
	return
}

func functionIdentity(pkg string, d *ast.FuncDecl) (id, name, recvType, recvVar string) {
	name = d.Name.Name
	if d.Recv == nil || len(d.Recv.List) == 0 {
		return pkg + "." + name, name, "", ""
	}
	recvType = exprName(d.Recv.List[0].Type)
	if recvType == "" {
		recvType = "unknown"
	}
	if len(d.Recv.List[0].Names) > 0 {
		recvVar = d.Recv.List[0].Names[0].Name
	}
	return pkg + "." + recvType + "." + name, name, recvType, recvVar
}

func analyzeFunctionBody(body *ast.BlockStmt, recvVar string) (complex, cognitive, nesting int, calls map[string]struct{}, fields map[string]struct{}) {
	calls = make(map[string]struct{})
	fields = make(map[string]struct{})
	complex = 1

	var walkExpr func(ast.Expr)
	walkExpr = func(e ast.Expr) {
		ast.Inspect(e, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.BinaryExpr:
				if x.Op == token.LAND || x.Op == token.LOR {
					complex++
					cognitive++
				}
			case *ast.CallExpr:
				if name := callName(x.Fun); name != "" {
					calls[name] = struct{}{}
				}
			case *ast.SelectorExpr:
				if recvVar != "" {
					if id, ok := x.X.(*ast.Ident); ok && id.Name == recvVar {
						fields[x.Sel.Name] = struct{}{}
					}
				}
			}
			return true
		})
	}

	var walkStmts func([]ast.Stmt, int)
	walkStmts = func(stmts []ast.Stmt, depth int) {
		if depth > nesting {
			nesting = depth
		}
		for _, stmt := range stmts {
			switch s := stmt.(type) {
			case *ast.IfStmt:
				complex++
				cognitive += 1 + depth
				if s.Init != nil {
					walkStmts([]ast.Stmt{s.Init}, depth)
				}
				walkExpr(s.Cond)
				walkStmts(s.Body.List, depth+1)
				if s.Else != nil {
					switch elseNode := s.Else.(type) {
					case *ast.BlockStmt:
						walkStmts(elseNode.List, depth+1)
					case *ast.IfStmt:
						walkStmts([]ast.Stmt{elseNode}, depth+1)
					}
				}
			case *ast.ForStmt:
				complex++
				cognitive += 1 + depth
				if s.Init != nil {
					walkStmts([]ast.Stmt{s.Init}, depth)
				}
				if s.Cond != nil {
					walkExpr(s.Cond)
				}
				if s.Post != nil {
					walkStmts([]ast.Stmt{s.Post}, depth)
				}
				walkStmts(s.Body.List, depth+1)
			case *ast.RangeStmt:
				complex++
				cognitive += 1 + depth
				walkExpr(s.X)
				walkStmts(s.Body.List, depth+1)
			case *ast.SwitchStmt:
				complex++
				cognitive += 1 + depth
				if s.Init != nil {
					walkStmts([]ast.Stmt{s.Init}, depth)
				}
				if s.Tag != nil {
					walkExpr(s.Tag)
				}
				for _, cc := range s.Body.List {
					cl := cc.(*ast.CaseClause)
					complex++
					cognitive += 1 + depth
					for _, expr := range cl.List {
						walkExpr(expr)
					}
					walkStmts(cl.Body, depth+1)
				}
			case *ast.TypeSwitchStmt:
				complex++
				cognitive += 1 + depth
				for _, cc := range s.Body.List {
					cl := cc.(*ast.CaseClause)
					complex++
					cognitive += 1 + depth
					walkStmts(cl.Body, depth+1)
				}
			case *ast.SelectStmt:
				complex++
				cognitive += 1 + depth
				for _, cc := range s.Body.List {
					cl := cc.(*ast.CommClause)
					complex++
					cognitive += 1 + depth
					walkStmts(cl.Body, depth+1)
				}
			case *ast.BlockStmt:
				walkStmts(s.List, depth+1)
			default:
				ast.Inspect(stmt, func(n ast.Node) bool {
					switch x := n.(type) {
					case ast.Expr:
						walkExpr(x)
					case *ast.CallExpr:
						if name := callName(x.Fun); name != "" {
							calls[name] = struct{}{}
						}
					}
					return true
				})
			}
		}
	}

	walkStmts(body.List, 0)
	return
}

func countStatements(body *ast.BlockStmt) int {
	count := 0
	ast.Inspect(body, func(n ast.Node) bool {
		if _, ok := n.(ast.Stmt); ok {
			count++
		}
		return true
	})
	return count
}

func callName(fun ast.Expr) string {
	switch v := fun.(type) {
	case *ast.Ident:
		return v.Name
	case *ast.SelectorExpr:
		prefix := exprName(v.X)
		if prefix == "" {
			return v.Sel.Name
		}
		return prefix + "." + v.Sel.Name
	default:
		return ""
	}
}

func ensureClass(classes map[string]*classMetric, key, pkg, name string) *classMetric {
	if c, ok := classes[key]; ok {
		return c
	}
	c := &classMetric{
		name:     name,
		fields:   make(map[string]struct{}),
		embedded: make([]string, 0, 2),
		methods:  make(map[string]*methodMetric),
		cboSet:   make(map[string]struct{}),
		pkg:      pkg,
		callSet:  make(map[string]struct{}),
		fieldSet: make(map[string]struct{}),
	}
	classes[key] = c
	return c
}

func computeOOMetrics(classes map[string]*classMetric, pkgImports map[string]map[string]struct{}) (wmcVals, ditVals, nocVals, cboVals, lcomVals, rfcVals []float64) {
	children := make(map[string]int)
	for _, c := range classes {
		for _, emb := range c.embedded {
			key := c.pkg + "." + emb
			children[key]++
		}
	}

	var ditDepth func(key string, seen map[string]struct{}) int
	ditDepth = func(key string, seen map[string]struct{}) int {
		if _, ok := seen[key]; ok {
			return 0
		}
		seen[key] = struct{}{}
		c, ok := classes[key]
		if !ok || len(c.embedded) == 0 {
			return 0
		}
		maxDepth := 0
		for _, emb := range c.embedded {
			d := 1 + ditDepth(c.pkg+"."+emb, cloneStringSet(seen))
			if d > maxDepth {
				maxDepth = d
			}
		}
		return maxDepth
	}

	for key, c := range classes {
		wmc := 0
		rfcSet := make(map[string]struct{})
		fieldMethodCount := make(map[string]int)
		for name, m := range c.methods {
			wmc += m.complex
			rfcSet[name] = struct{}{}
			for call := range m.calls {
				rfcSet[call] = struct{}{}
			}
			for f := range m.fields {
				if _, ok := c.fields[f]; ok {
					fieldMethodCount[f]++
				}
			}
		}

		lcom := 0.0
		M := float64(len(c.methods))
		F := float64(len(c.fields))
		if M > 1 && F > 0 {
			sumMf := 0.0
			for _, mf := range fieldMethodCount {
				sumMf += float64(mf)
			}
			lcom = (sumMf/F - M) / (1.0 - M)
			if lcom < 0 {
				lcom = 0
			}
			if lcom > 1 {
				lcom = 1
			}
		}

		d := ditDepth(key, map[string]struct{}{})
		noc := children[key]
		cbo := packageCoupling(pkgImports, c.pkg)
		rfc := len(rfcSet)

		wmcVals = append(wmcVals, float64(wmc))
		ditVals = append(ditVals, float64(d))
		nocVals = append(nocVals, float64(noc))
		cboVals = append(cboVals, float64(cbo))
		lcomVals = append(lcomVals, round2(lcom))
		rfcVals = append(rfcVals, float64(rfc))
	}
	return
}

func packageCoupling(pkgImports map[string]map[string]struct{}, pkg string) int {
	imports, ok := pkgImports[pkg]
	if !ok {
		return 0
	}
	set := make(map[string]struct{}, len(imports))
	for imp := range imports {
		if imp == pkg {
			continue
		}
		set[imp] = struct{}{}
	}
	return len(set)
}

func computeCoupling(pkgImports map[string]map[string]struct{}, modulePath string) (caVals, ceVals, instVals []float64) {
	afferent := make(map[string]int)
	efferent := make(map[string]int)

	for pkg, imports := range pkgImports {
		for imp := range imports {
			if modulePath == "" || !strings.HasPrefix(imp, modulePath) {
				continue
			}
			if imp == pkg {
				continue
			}
			efferent[pkg]++
			afferent[imp]++
		}
	}

	for pkg := range pkgImports {
		ca := afferent[pkg]
		ce := efferent[pkg]
		inst := 0.0
		if ca+ce > 0 {
			inst = float64(ce) / float64(ca+ce)
		}
		caVals = append(caVals, float64(ca))
		ceVals = append(ceVals, float64(ce))
		instVals = append(instVals, inst)
	}

	return
}

func computeDuplication(linesByFile map[string][]string, window, sloc int) duplicationMetrics {
	if window < 2 {
		window = 2
	}
	if len(linesByFile) == 0 {
		return duplicationMetrics{WindowSize: window}
	}
	counts := make(map[string]map[string]struct{})
	for file, lines := range linesByFile {
		if len(lines) < window {
			continue
		}
		for i := 0; i <= len(lines)-window; i++ {
			key := strings.Join(lines[i:i+window], "\n")
			fileSet, ok := counts[key]
			if !ok {
				fileSet = make(map[string]struct{})
				counts[key] = fileSet
			}
			fileSet[file] = struct{}{}
		}
	}

	dupWindows := 0
	for _, fileSet := range counts {
		if len(fileSet) > 1 {
			dupWindows += len(fileSet) - 1
		}
	}
	dupLOC := dupWindows * window
	if dupLOC > sloc {
		dupLOC = sloc
	}
	return duplicationMetrics{
		WindowSize:         window,
		DuplicateWindows:   dupWindows,
		ApproxDuplicateLOC: dupLOC,
		DuplicationPercent: percent(float64(dupLOC), float64(maxInt(1, sloc))),
	}
}

func parseCoverage(path string) coverageMetrics {
	if path == "" {
		return coverageMetrics{}
	}
	f, err := os.Open(path)
	if err != nil {
		return coverageMetrics{}
	}
	defer f.Close()

	total := 0
	covered := 0
	s := bufio.NewScanner(f)
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if line == "" || strings.HasPrefix(line, "mode:") {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) != 3 {
			continue
		}
		n, err1 := strconv.Atoi(parts[1])
		c, err2 := strconv.Atoi(parts[2])
		if err1 != nil || err2 != nil {
			continue
		}
		total += n
		if c > 0 {
			covered += n
		}
	}
	return coverageMetrics{
		Available:         true,
		TotalStatements:   total,
		CoveredStatements: covered,
		CodeCoverage:      percent(float64(covered), float64(maxInt(1, total))),
	}
}

func computeChangeMetrics(root string, files []string, window string) changeMetrics {
	m := changeMetrics{Window: window, TotalFiles: len(files)}
	currentFiles := make(map[string]struct{}, len(files))
	for _, f := range files {
		rel, err := filepath.Rel(root, f)
		if err != nil {
			continue
		}
		currentFiles[filepath.ToSlash(rel)] = struct{}{}
	}

	cmd := exec.Command("git", "log", "--since="+window, "--name-only", "--pretty=format:", "--", "*.go")
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		return m
	}
	m.Available = true

	fileSet := make(map[string]struct{})
	pkgCount := make(map[string]int)
	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || !strings.HasSuffix(line, ".go") {
			continue
		}
		if _, ok := currentFiles[filepath.ToSlash(line)]; !ok {
			continue
		}
		fileSet[line] = struct{}{}
		pkg := filepath.Dir(filepath.ToSlash(line))
		pkgCount[pkg]++
	}

	m.ChangedFiles = len(fileSet)
	m.ChangeInstability = percent(float64(m.ChangedFiles), float64(maxInt(1, m.TotalFiles)))

	maxPkg := ""
	maxCount := 0
	totalChanges := 0
	for pkg, c := range pkgCount {
		totalChanges += c
		if c > maxCount {
			maxCount = c
			maxPkg = pkg
		}
	}
	m.MostChangedPackage = maxPkg
	m.MostChangedPackagePct = percent(float64(maxCount), float64(maxInt(1, totalChanges)))
	return m
}

func calcDistribution(values []float64) distribution {
	if len(values) == 0 {
		return distribution{}
	}
	sort.Float64s(values)
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	idx := int(math.Ceil(0.95*float64(len(values)))) - 1
	if idx < 0 {
		idx = 0
	}
	if idx >= len(values) {
		idx = len(values) - 1
	}
	return distribution{
		Count: len(values),
		Min:   round2(values[0]),
		Mean:  round2(sum / float64(len(values))),
		P95:   round2(values[idx]),
		Max:   round2(values[len(values)-1]),
	}
}

func buildHotspots(functions map[string]*functionMetric, limit int) []hotspot {
	if limit <= 0 || len(functions) == 0 {
		return nil
	}

	items := make([]hotspot, 0, len(functions))
	for _, fn := range functions {
		items = append(items, hotspot{
			ID:         fn.id,
			Package:    fn.pkg,
			Name:       fn.name,
			Cyclomatic: fn.complex,
			Cognitive:  fn.cognitive,
			Nesting:    fn.nesting,
			Essential:  fn.essential,
			Pathologic: round2(fn.patho),
		})
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].Cognitive != items[j].Cognitive {
			return items[i].Cognitive > items[j].Cognitive
		}
		if items[i].Cyclomatic != items[j].Cyclomatic {
			return items[i].Cyclomatic > items[j].Cyclomatic
		}
		if items[i].Nesting != items[j].Nesting {
			return items[i].Nesting > items[j].Nesting
		}
		return items[i].ID < items[j].ID
	})

	if limit > len(items) {
		limit = len(items)
	}
	return items[:limit]
}

func maintainabilityIndex(halsteadVolume, cyclomaticTotal, sloc float64) float64 {
	if halsteadVolume <= 0 {
		halsteadVolume = 1
	}
	if sloc <= 0 {
		sloc = 1
	}
	mi := (171.0 - 5.2*math.Log(halsteadVolume) - 0.23*cyclomaticTotal - 16.2*math.Log(sloc)) * 100.0 / 171.0
	if mi < 0 {
		mi = 0
	}
	if mi > 100 {
		mi = 100
	}
	return round2(mi)
}

func readModulePath(goModPath string) (string, error) {
	b, err := os.ReadFile(goModPath)
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(string(b), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module")), nil
		}
	}
	return "", errors.New("module line not found")
}

func packageImportPath(modulePath, relDir string) string {
	if modulePath == "" {
		return relDir
	}
	if relDir == "" {
		return modulePath
	}
	return modulePath + "/" + filepath.ToSlash(relDir)
}

func exprName(e ast.Expr) string {
	switch v := e.(type) {
	case *ast.Ident:
		return v.Name
	case *ast.StarExpr:
		return exprName(v.X)
	case *ast.SelectorExpr:
		if left, ok := v.X.(*ast.Ident); ok {
			return left.Name + "." + v.Sel.Name
		}
		return v.Sel.Name
	default:
		return ""
	}
}

func hasDoc(cg *ast.CommentGroup) bool {
	return cg != nil && len(cg.List) > 0
}

func percent(part, total float64) float64 {
	if total <= 0 {
		return 0
	}
	return round2((part / total) * 100.0)
}

func sumFloat64(values []float64) float64 {
	s := 0.0
	for _, v := range values {
		s += v
	}
	return s
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}

func cloneStringSet(src map[string]struct{}) map[string]struct{} {
	dst := make(map[string]struct{}, len(src))
	for k := range src {
		dst[k] = struct{}{}
	}
	return dst
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
