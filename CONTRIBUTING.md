# Contributing to k8s-agent-stack

<!--
Copyright 2025 Raphaël MANSUY
Licensed under the Apache License, Version 2.0
https://www.apache.org/licenses/LICENSE-2.0
-->

Thank you for your interest in contributing to k8s-agent-stack! We welcome contributions from the community.

## How to Contribute

### 1. Report Bugs

If you find a bug, please create an issue with:
- Clear description of the problem
- Steps to reproduce
- Expected vs actual behavior
- Your environment (OS, Kubernetes version, etc.)

[Report a bug](https://github.com/raphaelmansuy/k8s-agent-stack/issues/new)

### 2. Suggest Features

We welcome feature requests! Please:
- Check if the feature is already requested
- Describe the use case clearly
- Explain why this benefits the project

[Request a feature](https://github.com/raphaelmansuy/k8s-agent-stack/issues/new)

### 3. Submit Pull Requests

#### Before You Start

1. **Fork the repository**
2. **Create a feature branch**: `git checkout -b feature/my-feature`
3. **Discuss major changes first**: Open an issue to discuss significant changes

#### Development Guidelines

**SOTA Go Development Environment:**
We use a "State of the Art" (SOTA) 2025 configuration for Go development. To set it up:

1.  **Install Go 1.24+**
2.  **Install Tools**: Run `make install-tools` from the root.
3.  **VS Code**: Use the provided `.vscode/settings.json` for real-time security scanning and strict formatting.

**Code Quality:**
- Follow existing code style
- Add tests for new features
- Update documentation
- Keep commits focused and atomic
- **Security**: Ensure `govulncheck` passes before submitting.
- **Formatting**: Use `gofumpt` (handled automatically by VS Code).

**Python Code:**
```python
# Use type hints
def process_request(message: str) -> dict:
    """Process a request and return response."""
    return {"result": message}

# Follow PEP 8
# Use docstrings
# Keep functions small and focused
```

**Documentation:**
- Update relevant docs in `docs/`
- Add inline code comments for complex logic
- Update README if adding features
- Include examples where helpful

#### Testing

```bash
# Run unit tests
cd kagent-adk-agent
make test

# Run integration tests
make test-integration

# Test deployment
make deploy-local
```

#### Commit Messages

Use clear, descriptive commit messages:

```
feat: add support for LangGraph agents
fix: resolve scale-to-zero cold start issue
docs: update deployment guide with traffic splitting
test: add integration tests for A2A protocol
```

Format: `type: description`

Types:
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation
- `test`: Tests
- `chore`: Maintenance
- `refactor`: Code restructuring

#### Pull Request Process

1. **Update your fork**: `git pull upstream main`
2. **Create PR** with clear description
3. **Link related issues**: Use "Fixes #123"
4. **Pass all checks**: CI must pass
5. **Request review**: Tag maintainers
6. **Address feedback**: Make requested changes
7. **Wait for approval**: Maintainer will merge

### 4. Improve Documentation

Documentation improvements are always welcome:
- Fix typos or unclear sections
- Add examples and tutorials
- Improve cross-linking
- Add diagrams and visualizations

See [docs/](docs/) for current documentation.

### 5. Add Agent Examples

We welcome new agent examples:

```bash
examples/
  my-agent/
    README.md        # How to use this agent
    agent.yaml       # Kagent manifest
    Dockerfile       # Container build
    requirements.txt # Dependencies
```

## Code of Conduct

### Our Standards

- **Be respectful**: Treat everyone with respect
- **Be collaborative**: Work together constructively
- **Be inclusive**: Welcome diverse perspectives
- **Be professional**: Focus on the work
- **Give credit**: Acknowledge others' contributions

### Unacceptable Behavior

- Harassment or discrimination
- Trolling or insulting comments
- Personal or political attacks
- Publishing private information
- Other unprofessional conduct

### Enforcement

Violations may result in temporary or permanent ban from the project.

## Questions?

- **Documentation**: Check [docs/](docs/)
- **Issues**: Search [existing issues](https://github.com/raphaelmansuy/k8s-agent-stack/issues)
- **Discussions**: Start a [discussion](https://github.com/raphaelmansuy/k8s-agent-stack/discussions)

## License

By contributing, you agree that your contributions will be licensed under the Apache License 2.0.

See [LICENSE](LICENSE) for details.

---

## Quick Reference

| Task | Command |
|------|---------|
| Run tests | `make test` |
| Build agent | `make build` |
| Deploy locally | `make deploy-local` |
| Format code | `make format` |
| Lint code | `make lint` |

Run `make help` for a full list of available commands.

---

**Thank you for contributing to k8s-agent-stack!**

[Back to README](README.md) • [Documentation](docs/) • [Examples](examples/)
