---
name: GitHub Actions Issue
about: Report a problem with GitHub Actions workflows
title: '[GitHub Actions] '
labels: ['github-actions', 'bug']
assignees: ''
---

## 🚨 GitHub Actions Issue

### **Workflow Affected**
- [ ] Test workflow
- [ ] Docker Build workflow
- [ ] Test (Simple) workflow
- [ ] Other: _______________

### **Error Details**

**Workflow Run URL:**
```
https://github.com/[username]/gorssag/actions/runs/[run-id]
```

**Error Message:**
```
[Paste the exact error message here]
```

**Failed Step:**
```
[Which step in the workflow failed?]
```

### **Environment**
- **Event Type:** `push` / `pull_request` / `release` / `manual`
- **Branch:** `main` / `feature-branch` / `other`
- **Commit SHA:** `[commit-hash]`
- **Runner:** `ubuntu-latest` / `other`

### **Expected Behavior**
```
[What should have happened?]
```

### **Actual Behavior**
```
[What actually happened?]
```

### **Additional Context**
- [ ] This is a new issue
- [ ] This worked before
- [ ] Related to recent changes: _______________
- [ ] Environment-specific issue

### **Screenshots**
If applicable, add screenshots to help explain your problem.

### **Reproduction Steps**
1. [Step 1]
2. [Step 2]
3. [Step 3]
4. See error

### **Troubleshooting Attempted**
- [ ] Checked workflow syntax
- [ ] Verified secrets are set
- [ ] Checked permissions
- [ ] Reviewed recent changes
- [ ] Other: _______________

---

**Note:** For Codecov rate limiting issues, the workflow is configured to continue on error. This is normal behavior for new repositories without a Codecov token. 