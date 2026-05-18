# Go Code Review Checklist

## Review Procedure

1. Run `gofmt -d .` and `go vet ./...` to catch mechanical issues first
2. Read the diff file-by-file; for each file, check the categories below in order
3. Flag issues with specific line references and the rule name
4. After reviewing all files, re-read flagged items to verify they're genuine issues
5. Summarize findings grouped by severity (must-fix, should-fix, nit)

---

## Formatting

- [ ] **gofmt**: Code is formatted with `gofmt` or `goimports`
- [ ] **goimports**: Imports are properly organized (stdlib, blank, external)

---

## Documentation

- [ ] **Comment sentences**: Comments are full sentences ending with a period
- [ ] **Doc comments**: All exported names have doc comments
- [ ] **Package comments**: Package comment appears adjacent to package clause

---

## Error Handling

- [ ] **Handle errors**: No discarded errors with `_`
- [ ] **Error strings**: Lowercase, no punctuation
- [ ] **Indent error flow**: Handle errors first and return

---

## Naming

- [ ] **MixedCaps**: Use MixedCaps, never underscores
- [ ] **Initialisms**: Keep consistent case (URL, ID, HTTP)
- [ ] **Variable names**: Short names for limited scope
- [ ] **Receiver names**: One or two letter abbreviation

---

## Concurrency

- [ ] **Goroutine lifetimes**: Clear when/whether goroutines exit
- [ ] **Contexts**: First parameter; pass even if you think you don't need to

---

## Interfaces

- [ ] **Interface location**: Define in consumer package, not implementor
- [ ] **No premature interfaces**: Don't define before used
- [ ] **Receiver type**: Use pointer if mutating

---

## Security

- [ ] **Crypto rand**: Use crypto/rand for keys, not math/rand
- [ ] **Don't panic**: Use error returns for normal error handling

---

## Declarations and Initialization

- [ ] **Group similar**: Related var/const/type in parenthesized blocks
- [ ] **Reduce scope**: Move declarations close to usage

---

## Logging

- [ ] **Use slog**: New code uses log/slog for operational logging
- [ ] **No secrets in logs**: Credentials are never logged

---

## Imports

- [ ] **Import groups**: Standard library first, then blank line, then external packages
- [ ] **Import renaming**: Avoid unless collision

---

## Testing

- [ ] **Examples**: Include runnable Example functions
- [ ] **Useful test failures**: Messages include what was wrong, inputs, got, and want

---

## Automated Checks

```bash
gofmt -l . && go vet ./... && golangci-lint run ./...
```

Fix any issues before proceeding to the checklist above.