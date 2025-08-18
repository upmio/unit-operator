# Contributing to Unit Operator

Thank you for your interest in contributing to the Unit Operator project! This document provides guidelines and instructions for contributors.

## 🤝 How to Contribute

### Reporting Bugs

If you find a bug, please create an issue on GitHub with the following information:

- **Clear description** of the problem
- **Steps to reproduce** the issue
- **Expected behavior** vs. **actual behavior**
- **Environment information**:
  - Kubernetes version
  - Unit Operator version
  - Database type and version
  - Operating system

### Suggesting Features

We welcome feature suggestions! Please include:

- **Clear description** of the feature
- **Use case** and motivation
- **Proposed implementation** (if known)
- **Alternative solutions** considered

### Code Contributions

#### Development Setup

1. **Fork the repository**
2. **Clone your fork**:
   ```bash
   git clone https://github.com/your-username/unit-operator.git
   cd unit-operator
   ```

3. **Set up development environment**:
   ```bash
   # Install Go 1.23+
   # Install required tools
   make install-tools
   
   # Install dependencies
   go mod download
   
   # Run tests to verify setup
   make test
   ```

#### Development Workflow

1. **Create a feature branch**:
   ```bash
   git checkout -b feature/your-feature-name
   ```

2. **Make your changes**:
   - Follow the existing code style
   - Add tests for new functionality
   - Update documentation as needed

3. **Run local checks**:
   ```bash
   # Format code
   make fmt
   
   # Run vet
   make vet
   
   # Run tests
   make test
   
   # Run linting
   make lint
   
   # Check coverage
   make check-coverage
   ```

4. **Commit your changes**:
   ```bash
   git add .
   git commit -m "feat: add your feature description"
   ```

5. **Push to your fork**:
   ```bash
   git push origin feature/your-feature-name
   ```

6. **Create a Pull Request**:
   - Fill in the PR template
   - Link to relevant issues
   - Ensure all checks pass

#### Commit Message Convention

We use [Conventional Commits](https://www.conventionalcommits.org/) format:

```
<type>[optional scope]: <description>

[optional body]

[optional footer(s)]
```

**Types:**
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `style`: Code style changes
- `refactor`: Code refactoring
- `test`: Test changes
- `chore`: Build process or auxiliary tool changes

**Example:**
```
feat(unitset): add PostgreSQL replication support

- Add streaming replication configuration
- Include health checks for replication status
- Update documentation with replication examples

Closes #123
```

## 📝 Code Standards

### Go Code Style

- Follow [Go's official formatting guidelines](https://golang.org/doc/effective_go.html)
- Use `gofmt` for code formatting
- Write clear, descriptive comments
- Use meaningful variable and function names

### Testing Requirements

- **Unit tests**: Required for all new functionality
- **Integration tests**: For complex features
- **Test coverage**: Maintain minimum coverage threshold
- **Test naming**: Use descriptive test names

### Documentation

- **API documentation**: Update for API changes
- **Examples**: Provide usage examples
- **README**: Update for major features
- **Comments**: Add godoc comments for exported functions

## 🔧 Development Tools

### Required Tools

- **Go**: 1.23+
- **Docker**: For building and testing
- **Kubectl**: For Kubernetes interaction
- **Helm**: For chart management

### Makefile Commands

```bash
# Build and Development
make build          # Build the manager binary
make run            # Run the controller locally
make manifests      # Generate CRDs and manifests
make generate       # Generate code
make docker-build   # Build Docker image
make docker-push    # Push Docker image

# Testing and Quality
make test           # Run unit tests
make test-e2e       # Run e2e tests
make test-report    # Generate test coverage
make check-coverage # Check test coverage
make fmt            # Format code
make vet            # Run go vet
make lint           # Run linter
make lint-fix       # Run linter with auto-fix

# Deployment
make install        # Install CRDs
make uninstall      # Uninstall CRDs
make deploy         # Deploy to cluster
make undeploy       # Undeploy from cluster
make bundle         # Generate OLM bundle
```

## 🧪 Testing Guidelines

### Unit Tests

- Write tests for all new functions
- Mock external dependencies
- Test edge cases and error conditions
- Use table-driven tests when appropriate

### Integration Tests

- Test interaction with Kubernetes
- Use test clusters (Kind, Minikube)
- Clean up resources after tests

### E2E Tests

- Test complete workflows
- Use real Kubernetes clusters
- Test upgrade scenarios

## 📋 Pull Request Process

### Before Submitting

1. **Search existing PRs** to avoid duplicates
2. **Update documentation** for your changes
3. **Add tests** for new functionality
4. **Run all checks** locally
5. **Ensure your code builds** successfully

### PR Review Process

1. **Automated checks** must pass
2. **Code review** by maintainers
3. **Testing** by maintainers
4. **Approval** before merging

### Merge Criteria

- **All checks** pass
- **At least one approval** from maintainer
- **Documentation** updated
- **Tests** added/updated
- **Backward compatibility** maintained

## 🏗️ Architecture Guidelines

### Project Structure

```
pkg/
├── api/          # API definitions
├── controller/   # Controllers
├── agent/        # Agent implementation
├── webhook/      # Webhooks
├── utils/        # Utility functions
└── certs/        # Certificate management
```

### Design Principles

- **Kubernetes-native**: Follow Kubernetes patterns
- **Declarative**: Use desired state configuration
- **Idempotent**: Operations should be repeatable
- **Observable**: Include metrics and logging
- **Secure**: Follow security best practices

## 🌍 Community Guidelines

### Communication

- **GitHub Issues**: For bugs and feature requests
- **GitHub Discussions**: For general questions
- **Pull Requests**: For code contributions
- **Email**: For private communications

### Getting Help

- **Documentation**: Check the README and API docs
- **Examples**: Look at the examples directory
- **Issues**: Search existing issues
- **Discussions**: Ask questions in GitHub Discussions

## 📈 Release Process

### Versioning

We use [Semantic Versioning](https://semver.org/):

- **Major versions**: Breaking changes
- **Minor versions**: New features
- **Patch versions**: Bug fixes

### Release Checklist

1. **Update version** in relevant files
2. **Update CHANGELOG**
3. **Create release tag**
4. **Build and publish artifacts**
5. **Update documentation**
6. **Announce release**

## 🏆 Recognition

Contributors will be recognized in:

- **RELEASES.md**: Release notes
- **AUTHORS.md**: Contributor list
- **GitHub**: Contributors section
- **Blog posts**: For significant contributions

## 📞 Contact

- **Maintainers**: See MAINTAINERS.md
- **Email**: unit-operator@example.com
- **Discussions**: [GitHub Discussions](https://github.com/upmio/unit-operator/discussions)

---

Thank you for contributing to Unit Operator! 🎉

## 📚 Additional Resources

- [Kubernetes Documentation](https://kubernetes.io/docs/)
- [Operator Framework](https://operatorframework.io/)
- [Go Documentation](https://golang.org/doc/)
- [Docker Documentation](https://docs.docker.com/)

---

<div align="center">
  <p>
    <img src="https://img.icons8.com/color/48/000000/kubernetes.png" alt="Kubernetes" width="32" height="32">
    <img src="https://img.icons8.com/color/48/000000/database.png" alt="Database" width="32" height="32">
    <img src="https://img.icons8.com/color/48/000000/code.png" alt="Code" width="32" height="32">
  </p>
  <p><strong>Happy Contributing!</strong></p>
</div>