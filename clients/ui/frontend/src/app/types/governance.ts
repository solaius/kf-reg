/** Governance types matching the catalog server governance API schema. */

export type LifecycleState = 'draft' | 'approved' | 'deprecated' | 'archived';

export type RiskLevel = 'low' | 'medium' | 'high' | 'critical';

export type SLATier = 'gold' | 'silver' | 'bronze' | 'none';

export type OwnerInfo = {
  principal?: string;
  displayName?: string;
  email?: string;
};

export type TeamInfo = {
  name?: string;
  id?: string;
};

export type SLAInfo = {
  tier?: SLATier;
  responseHours?: number;
};

export type RiskInfo = {
  level?: RiskLevel;
  categories?: string[];
};

export type IntendedUse = {
  summary?: string;
  environments?: string[];
  restrictions?: string[];
};

export type ComplianceInfo = {
  tags?: string[];
  controls?: string[];
};

export type LifecycleInfo = {
  state: LifecycleState;
  reason?: string;
  changedBy?: string;
  changedAt?: string;
};

export type AuditMetadata = {
  lastReviewedAt?: string;
  reviewCadenceDays?: number;
};

export type GovernanceOverlay = {
  owner?: OwnerInfo;
  team?: TeamInfo;
  sla?: SLAInfo;
  risk?: RiskInfo;
  intendedUse?: IntendedUse;
  compliance?: ComplianceInfo;
  lifecycle?: LifecycleInfo;
  audit?: AuditMetadata;
};

export type AssetRef = {
  plugin: string;
  kind: string;
  name: string;
};

export type GovernanceResponse = {
  assetRef: AssetRef;
  governance: GovernanceOverlay;
};

export type AuditEvent = {
  id: string;
  correlationId: string;
  eventType: string;
  actor: string;
  assetUid: string;
  versionId?: string;
  action?: string;
  outcome: string;
  reason?: string;
  oldValue?: Record<string, unknown>;
  newValue?: Record<string, unknown>;
  metadata?: Record<string, unknown>;
  createdAt: string;
};

export type AuditEventList = {
  events: AuditEvent[];
  nextPageToken?: string;
  totalSize: number;
};

export type ActionResult = {
  action: string;
  status: string;
  message?: string;
  data?: Record<string, unknown>;
};

export type VersionResponse = {
  versionId: string;
  versionLabel: string;
  createdAt: string;
  createdBy: string;
  contentDigest?: string;
  provenance?: ProvenanceInfo;
};

export type VersionListResponse = {
  versions: VersionResponse[];
  nextPageToken?: string;
  totalSize: number;
};

export type ProvenanceInfo = {
  sourceType?: string;
  sourceUri?: string;
  sourceId?: string;
  revisionId?: string;
  observedAt?: string;
  integrity?: IntegrityInfo;
};

export type IntegrityInfo = {
  verified: boolean;
  method?: string;
  details?: string;
};

export type BindingResponse = {
  environment: string;
  versionId: string;
  boundAt: string;
  boundBy: string;
  previousVersionId?: string;
};

export type BindingsResponse = {
  bindings: BindingResponse[];
};

export type ApprovalRequest = {
  id: string;
  assetRef: AssetRef;
  action: string;
  actionParams?: Record<string, unknown>;
  policyId: string;
  requiredCount: number;
  status: 'pending' | 'approved' | 'denied' | 'canceled' | 'expired';
  requester: string;
  reason?: string;
  decisions?: ApprovalDecision[];
  resolvedAt?: string;
  resolvedBy?: string;
  resolutionNote?: string;
  expiresAt?: string;
  createdAt: string;
};

export type ApprovalDecision = {
  id: string;
  requestId: string;
  reviewer: string;
  verdict: 'approve' | 'deny';
  comment?: string;
  createdAt: string;
};

export type ApprovalRequestList = {
  requests: ApprovalRequest[];
  nextPageToken?: string;
  totalSize: number;
};

export type GovernanceCapabilities = {
  supported: boolean;
  lifecycle?: {
    states: string[];
    defaultState: string;
  };
  versioning?: {
    enabled: boolean;
    environments: string[];
  };
  approvals?: {
    enabled: boolean;
  };
  provenance?: {
    enabled: boolean;
  };
};
