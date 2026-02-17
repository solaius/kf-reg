# 05 Approval Workflows
**Date**: 2026-02-17  
**Status**: Spec

## Purpose
Introduce lightweight approvals that can gate lifecycle transitions and promotions by asset type, action, and labels.

This is not a full workflow engine. It is a consistent, auditable gating layer.

## Approval policy configuration
Config file:
- catalog/config/approval-policies.yaml

Example:
```yaml
apiVersion: catalog.kubeflow.org/v1alpha1
kind: ApprovalPolicies
defaults:
  requireOwnerForApproval: true
rules:
  - name: "Agents require approval"
    selector:
      kinds: ["Agent"]
    gates:
      - action: "lifecycle.setState"
        when:
          toState: "approved"
        approvalsRequired: 1
        approverRoles: ["operator","approver"]

  - name: "High risk to prod requires 2 approvals"
    selector:
      any:
        - labels:
            risk: "high"
        - governance:
            risk.level: "high"
    gates:
      - action: "promotion.bind"
        when:
          environment: "prod"
        approvalsRequired: 2
        approverRoles: ["approver","security-approver"]

  - name: "Archived assets cannot be promoted"
    selector:
      matchAll: true
    gates:
      - action: "promotion.bind"
        denyWhen:
          lifecycle.state: "archived"
```

## Approval request model
Statuses:
- pending, approved, rejected, expired, cancelled

Fields:
- requestId
- asset uid, kind, name, versionId
- action and params
- requestedBy, createdAt
- decisions list (approver, decision, time, comment)

## Server behavior
- If an action is gated:
  - return approval-required response with requestId
- Approvers approve via UI or CLI
- When required approvals are met:
  - execute action atomically
  - write audit event including approvals

## Definition of Done
- Approval policies loaded and applied
- At least 2 gate rules enforced (by kind and by risk or label)
- UI and CLI support request, approve, reject
- Full audit trail for approvals and execution

## Acceptance Criteria
- Attempt to approve an Agent without required governance.owner fails with validation error
- Promote high-risk asset to prod creates approval request
- After 2 approvals, promotion executes and binding updates
- Rejection prevents execution and recorded in audit history
