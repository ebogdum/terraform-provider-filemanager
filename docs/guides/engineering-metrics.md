<!-- SPDX-License-Identifier: MIT -->

# Engineering Metrics

This project includes a metrics CLI to baseline and track code quality, complexity, maintainability, and delivery signals.

## Run

```bash
# 1) Generate coverage profile
GOCACHE=$(pwd)/.cache/go-build go test ./... -coverprofile=coverage.out

# 2) Generate metrics JSON
GOCACHE=$(pwd)/.cache/go-build go run ./tools/quality_metrics.go -root . -coverage-file ./coverage.out > metrics.json
```

## Metric Coverage

The generated JSON includes the following metrics families:

- Complexity: Cyclomatic, Cognitive, Essential (proxy), Pathological (proxy), Nesting Depth
- Size: LOC, SLOC, LLOC, Comment Density
- Halstead: Volume, Length, Difficulty, Effort, Vocabulary
- OO: WMC, DIT (embedding proxy), NOC (embedding proxy), CBO (AST coupling proxy), LCOM (method/field usage proxy), RFC
- Maintainability: Maintainability Index (MI)
- Quality and Risk: Technical Debt (proxy), Duplication Percentage (sliding-window proxy), Documentation Threshold/Coverage
- Testing and Change: Code Coverage, Change Instability (git-window based)
- Coupling and Flow: Afferent Coupling, Efferent Coupling, Fan-in, Fan-out

## Notes

- Go does not have classical inheritance; DIT/NOC are approximated via struct embedding depth and embedding children.
- Essential and pathological complexity are heuristic proxies derived from control-flow and nesting.
- CBO/LCOM/RFC are static AST approximations and can undercount runtime or reflection-driven behavior.
- Duplication uses normalized multi-line windows and is intended for trend tracking, not exact clone detection.
