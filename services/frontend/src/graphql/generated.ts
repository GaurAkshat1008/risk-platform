/* eslint-disable @typescript-eslint/no-explicit-any */
/**
 * AUTO-GENERATED — run `npm run codegen` to regenerate.
 *
 * This stub provides type signatures so the project compiles
 * before graphql-codegen has been executed.
 */
import { gql } from '@apollo/client';
import * as Apollo from '@apollo/client';

export type Scalars = {
  ID: string;
  String: string;
  Boolean: boolean;
  Int: number;
  Float: number;
  Time: string;
};

/* ─── Enums ─── */
export type PaymentStatus = 'RECEIVED' | 'DUPLICATE' | 'REJECTED';
export type Outcome = 'APPROVE' | 'FLAG' | 'REVIEW' | 'BLOCK';
export type CaseStatus = 'OPEN' | 'IN_REVIEW' | 'RESOLVED' | 'ESCALATED';
export type CasePriority = 'LOW' | 'MEDIUM' | 'HIGH' | 'CRITICAL';
export type GuardType = 'ROLE_REQUIRED' | 'CONDITION';
export type RuleAction = 'APPROVE' | 'FLAG' | 'REVIEW' | 'BLOCK';
export type TenantStatus = 'ACTIVE' | 'INACTIVE' | 'ONBOARDING';
export type NotificationChannel = 'EMAIL' | 'WEBHOOK' | 'SLACK';
export type NotificationStatus = 'PENDING' | 'DELIVERED' | 'FAILED' | 'RETRYING';

/* ─── Object types ─── */
export type PaymentEvent = {
  __typename?: 'PaymentEvent';
  id: string;
  idempotencyKey: string;
  tenantId: string;
  amount: number;
  currency: string;
  source: string;
  destination: string;
  status: PaymentStatus;
  receivedAt: string;
};

export type IngestionResult = {
  __typename?: 'IngestionResult';
  eventId: string;
  status: PaymentStatus;
  reason: string;
};

export type Decision = {
  __typename?: 'Decision';
  id: string;
  paymentEventId: string;
  tenantId: string;
  outcome: Outcome;
  reasonCodes: string[];
  confidenceScore: number;
  overridden: boolean;
  latencyMs: number;
  createdAt: string;
};

export type DecisionConnection = {
  __typename?: 'DecisionConnection';
  decisions: Decision[];
  total: number;
};

export type RuleContribution = {
  __typename?: 'RuleContribution';
  ruleId: string;
  ruleName: string;
  matched: boolean;
  action: string;
  reason: string;
};

export type FeatureValue = {
  __typename?: 'FeatureValue';
  name: string;
  value: string;
};

export type Explanation = {
  __typename?: 'Explanation';
  id: string;
  decisionId: string;
  tenantId: string;
  paymentEventId: string;
  outcome: Outcome;
  confidenceScore: number;
  ruleContributions: RuleContribution[];
  featureValues: FeatureValue[];
  narrative: string;
  policyVersion: string;
  generatedAt: string;
};

export type Case = {
  __typename?: 'Case';
  id: string;
  decisionId: string;
  tenantId: string;
  assigneeId: string;
  status: CaseStatus;
  priority: CasePriority;
  paymentEventId: string;
  outcome: string;
  slaDeadline: string;
  createdAt: string;
  updatedAt: string;
};

export type CaseConnection = {
  __typename?: 'CaseConnection';
  cases: Case[];
  nextPageToken: string;
};

export type Rule = {
  __typename?: 'Rule';
  id: string;
  tenantId: string;
  name: string;
  version: number;
  action: RuleAction;
  priority: number;
  enabled: boolean;
  createdAt: string;
  updatedAt: string;
};

export type RuleEvalResult = {
  __typename?: 'RuleEvalResult';
  ruleId: string;
  ruleName: string;
  matched: boolean;
  action: RuleAction;
  reason: string;
};

export type WorkflowTransition = {
  __typename?: 'WorkflowTransition';
  fromState: string;
  toState: string;
  requiredRole: string;
  guards: WorkflowGuard[];
};

export type WorkflowGuard = {
  __typename?: 'WorkflowGuard';
  type: GuardType;
  role: string;
  condition: string;
};

export type WorkflowTemplate = {
  __typename?: 'WorkflowTemplate';
  id: string;
  tenantId: string;
  name: string;
  version: number;
  states: string[];
  transitions: WorkflowTransition[];
  createdAt: string;
  updatedAt: string;
};

export type TransitionResult = {
  __typename?: 'TransitionResult';
  allowed: boolean;
  reason: string;
};

export type Tenant = {
  __typename?: 'Tenant';
  id: string;
  name: string;
  status: TenantStatus;
  createdAt: string;
  config: TenantConfig;
};

export type TenantConfig = {
  __typename?: 'TenantConfig';
  ruleSetId: string;
  workflowTemplateId: string;
  featureFlags: FeatureFlag[];
  version: number;
};

export type FeatureFlag = {
  __typename?: 'FeatureFlag';
  key: string;
  enabled: boolean;
  rolloutPercentage: number;
};

export type AuditEvent = {
  __typename?: 'AuditEvent';
  id: string;
  tenantId: string;
  actorId: string;
  action: string;
  resourceType: string;
  resourceId: string;
  sourceTopic: string;
  hash: string;
  previousHash: string;
  occurredAt: string;
};

export type AuditConnection = {
  __typename?: 'AuditConnection';
  events: AuditEvent[];
  nextPageToken: string;
};

export type ChainVerification = {
  __typename?: 'ChainVerification';
  valid: boolean;
  eventsChecked: number;
  brokenAtId: string;
};

export type Notification = {
  __typename?: 'Notification';
  id: string;
  tenantId: string;
  type: string;
  recipient: string;
  channel: NotificationChannel;
  status: NotificationStatus;
  attempts: number;
  createdAt: string;
};

export type NotificationPreference = {
  __typename?: 'NotificationPreference';
  tenantId: string;
  channel: NotificationChannel;
  eventType: string;
  enabled: boolean;
  config: string;
};

export type NotificationSendResult = {
  __typename?: 'NotificationSendResult';
  notificationId: string;
  status: NotificationStatus;
};

export type LogEntry = {
  __typename?: 'LogEntry';
  id: string;
  service: string;
  severity: string;
  message: string;
  traceId: string;
  spanId: string;
  tenantId: string;
  environment: string;
  timestamp: string;
};

export type LogConnection = {
  __typename?: 'LogConnection';
  entries: LogEntry[];
  nextPageToken: string;
};

export type TraceSpan = {
  __typename?: 'TraceSpan';
  traceId: string;
  spanId: string;
  parentSpanId: string;
  service: string;
  operation: string;
  durationMs: number;
  status: string;
  startTime: string;
};

export type SLOStatus = {
  __typename?: 'SLOStatus';
  service: string;
  errorRate: number;
  p50LatencyMs: number;
  p95LatencyMs: number;
  p99LatencyMs: number;
  availability: number;
  window: string;
};

export type Alert = {
  __typename?: 'Alert';
  name: string;
  service: string;
  severity: string;
  state: string;
  summary: string;
  firedAt: string;
};

/* ─── Input types ─── */
export type IngestPaymentInput = {
  idempotencyKey: string;
  tenantId: string;
  amount: number;
  currency: string;
  source: string;
  destination: string;
};

export type OverrideDecisionInput = {
  decisionId: string;
  analystId: string;
  newOutcome: Outcome;
  reason: string;
};

export type CreateCaseInput = {
  decisionId: string;
  tenantId: string;
  paymentEventId: string;
  outcome: string;
  priority: CasePriority;
};

export type AssignCaseInput = {
  caseId: string;
  assigneeId: string;
  actorId: string;
};

export type UpdateCaseStatusInput = {
  caseId: string;
  status: CaseStatus;
  actorId: string;
  notes: string;
};

export type EscalateCaseInput = {
  caseId: string;
  actorId: string;
  reason: string;
};

export type LogQueryInput = {
  service?: string | null;
  severity?: string | null;
  traceId?: string | null;
  tenantId?: string | null;
  messageContains?: string | null;
  fromTime?: string | null;
  toTime?: string | null;
  pageSize?: number | null;
  pageToken?: string | null;
};

export type AuditQueryInput = {
  tenantId: string;
  actorId?: string | null;
  resourceType?: string | null;
  resourceId?: string | null;
  action?: string | null;
  fromTime?: string | null;
  toTime?: string | null;
  pageSize?: number | null;
  pageToken?: string | null;
};

export type CreateWorkflowTemplateInput = {
  tenantId: string;
  name: string;
  states: string[];
  transitions: TransitionInput[];
};

export type UpdateWorkflowTemplateInput = {
  templateId: string;
  tenantId: string;
  states: string[];
  transitions: TransitionInput[];
};

export type TransitionInput = {
  fromState: string;
  toState: string;
  requiredRole: string;
};

export type EvaluateTransitionInput = {
  templateId: string;
  tenantId: string;
  fromState: string;
  toState: string;
  actorRole: string;
};

export type CreateRuleInput = {
  tenantId: string;
  name: string;
  expression: string;
  action: RuleAction;
  priority: number;
};

export type UpdateRuleInput = {
  ruleId: string;
  tenantId: string;
  expression: string;
  action: RuleAction;
  priority: number;
  enabled: boolean;
};

export type DeleteRuleInput = {
  ruleId: string;
  tenantId: string;
};

export type SimulateRuleInput = {
  tenantId: string;
  ruleId: string;
  expression: string;
  action: RuleAction;
  paymentEventId: string;
  amount: number;
  currency: string;
  source: string;
  destination: string;
};

export type CreateTenantInput = {
  name: string;
  ruleSetId: string;
  workflowTemplateId: string;
};

export type UpdateTenantRuleConfigInput = {
  tenantId: string;
  ruleSetId: string;
};

export type UpdateTenantWorkflowConfigInput = {
  tenantId: string;
  workflowTemplateId: string;
};

export type SendNotificationInput = {
  tenantId: string;
  type: string;
  recipient: string;
  channel: NotificationChannel;
  payload: string;
};

export type UpdateNotificationPreferencesInput = {
  tenantId: string;
  channel: NotificationChannel;
  eventType: string;
  enabled: boolean;
  config: string;
};

/* ─── Documents ─── */

/* -- Merchant -- */
export const IngestPaymentDocument = gql`
  mutation IngestPayment($input: IngestPaymentInput!) {
    ingestPayment(input: $input) { eventId status reason }
  }
`;
export function useIngestPaymentMutation(options?: Apollo.MutationHookOptions<{ ingestPayment: IngestionResult }, { input: IngestPaymentInput }>) {
  return Apollo.useMutation<{ ingestPayment: IngestionResult }, { input: IngestPaymentInput }>(IngestPaymentDocument, options);
}

export const PaymentEventDocument = gql`
  query PaymentEvent($tenantId: ID!, $idempotencyKey: String!) {
    paymentEvent(tenantId: $tenantId, idempotencyKey: $idempotencyKey) {
      id idempotencyKey tenantId amount currency source destination status receivedAt
    }
  }
`;
export function usePaymentEventQuery(options: Apollo.QueryHookOptions<{ paymentEvent: PaymentEvent | null }, { tenantId: string; idempotencyKey: string }>) {
  return Apollo.useQuery<{ paymentEvent: PaymentEvent | null }, { tenantId: string; idempotencyKey: string }>(PaymentEventDocument, options);
}
export function usePaymentEventLazyQuery(options?: Apollo.LazyQueryHookOptions<{ paymentEvent: PaymentEvent | null }, { tenantId: string; idempotencyKey: string }>) {
  return Apollo.useLazyQuery<{ paymentEvent: PaymentEvent | null }, { tenantId: string; idempotencyKey: string }>(PaymentEventDocument, options);
}

export const DecisionsDocument = gql`
  query Decisions($tenantId: ID!, $page: Int, $pageSize: Int, $outcomeFilter: String) {
    decisions(tenantId: $tenantId, page: $page, pageSize: $pageSize, outcomeFilter: $outcomeFilter) {
      decisions { id paymentEventId tenantId outcome reasonCodes confidenceScore overridden latencyMs createdAt }
      total
    }
  }
`;
export function useDecisionsQuery(options: Apollo.QueryHookOptions<{ decisions: DecisionConnection }, { tenantId: string; page?: number; pageSize?: number; outcomeFilter?: string }>) {
  return Apollo.useQuery<{ decisions: DecisionConnection }, { tenantId: string; page?: number; pageSize?: number; outcomeFilter?: string }>(DecisionsDocument, options);
}

export const DecisionDocument = gql`
  query Decision($tenantId: ID!, $paymentEventId: ID!) {
    decision(tenantId: $tenantId, paymentEventId: $paymentEventId) {
      id paymentEventId tenantId outcome reasonCodes confidenceScore overridden latencyMs createdAt
    }
  }
`;
export function useDecisionQuery(options: Apollo.QueryHookOptions<{ decision: Decision | null }, { tenantId: string; paymentEventId: string }>) {
  return Apollo.useQuery<{ decision: Decision | null }, { tenantId: string; paymentEventId: string }>(DecisionDocument, options);
}

export const ExplanationDocument = gql`
  query Explanation($tenantId: ID!, $paymentEventId: ID!) {
    explanation(tenantId: $tenantId, paymentEventId: $paymentEventId) {
      id decisionId tenantId paymentEventId outcome confidenceScore
      ruleContributions { ruleId ruleName matched action reason }
      featureValues { name value }
      narrative policyVersion generatedAt
    }
  }
`;
export function useExplanationQuery(options: Apollo.QueryHookOptions<{ explanation: Explanation | null }, { tenantId: string; paymentEventId: string }>) {
  return Apollo.useQuery<{ explanation: Explanation | null }, { tenantId: string; paymentEventId: string }>(ExplanationDocument, options);
}

export const OverrideDecisionDocument = gql`
  mutation OverrideDecision($input: OverrideDecisionInput!) {
    overrideDecision(input: $input)
  }
`;
export function useOverrideDecisionMutation(options?: Apollo.MutationHookOptions<{ overrideDecision: string }, { input: OverrideDecisionInput }>) {
  return Apollo.useMutation<{ overrideDecision: string }, { input: OverrideDecisionInput }>(OverrideDecisionDocument, options);
}

/* -- Analyst -- */
export const CasesDocument = gql`
  query Cases($tenantId: ID!, $status: CaseStatus, $assigneeId: ID, $pageSize: Int, $pageToken: String) {
    cases(tenantId: $tenantId, status: $status, assigneeId: $assigneeId, pageSize: $pageSize, pageToken: $pageToken) {
      cases { id decisionId tenantId assigneeId status priority paymentEventId outcome slaDeadline createdAt updatedAt }
      nextPageToken
    }
  }
`;
export function useCasesQuery(options: Apollo.QueryHookOptions<{ cases: CaseConnection }, { tenantId: string; status?: CaseStatus; assigneeId?: string; pageSize?: number; pageToken?: string }>) {
  return Apollo.useQuery<{ cases: CaseConnection }, { tenantId: string; status?: CaseStatus; assigneeId?: string; pageSize?: number; pageToken?: string }>(CasesDocument, options);
}

export const CaseDetailDocument = gql`
  query CaseDetail($caseId: ID!) {
    case(caseId: $caseId) {
      id decisionId tenantId assigneeId status priority paymentEventId outcome slaDeadline createdAt updatedAt
    }
  }
`;
export function useCaseDetailQuery(options: Apollo.QueryHookOptions<{ case: Case | null }, { caseId: string }>) {
  return Apollo.useQuery<{ case: Case | null }, { caseId: string }>(CaseDetailDocument, options);
}

export const CreateCaseDocument = gql`
  mutation CreateCase($input: CreateCaseInput!) {
    createCase(input: $input) { id decisionId tenantId status priority createdAt }
  }
`;
export function useCreateCaseMutation(options?: Apollo.MutationHookOptions<{ createCase: Case }, { input: CreateCaseInput }>) {
  return Apollo.useMutation<{ createCase: Case }, { input: CreateCaseInput }>(CreateCaseDocument, options);
}

export const AssignCaseDocument = gql`
  mutation AssignCase($input: AssignCaseInput!) {
    assignCase(input: $input) { id assigneeId status updatedAt }
  }
`;
export function useAssignCaseMutation(options?: Apollo.MutationHookOptions<{ assignCase: Case }, { input: AssignCaseInput }>) {
  return Apollo.useMutation<{ assignCase: Case }, { input: AssignCaseInput }>(AssignCaseDocument, options);
}

export const UpdateCaseStatusDocument = gql`
  mutation UpdateCaseStatus($input: UpdateCaseStatusInput!) {
    updateCaseStatus(input: $input) { id status updatedAt }
  }
`;
export function useUpdateCaseStatusMutation(options?: Apollo.MutationHookOptions<{ updateCaseStatus: Case }, { input: UpdateCaseStatusInput }>) {
  return Apollo.useMutation<{ updateCaseStatus: Case }, { input: UpdateCaseStatusInput }>(UpdateCaseStatusDocument, options);
}

export const EscalateCaseDocument = gql`
  mutation EscalateCase($input: EscalateCaseInput!) {
    escalateCase(input: $input) { id status priority updatedAt }
  }
`;
export function useEscalateCaseMutation(options?: Apollo.MutationHookOptions<{ escalateCase: Case }, { input: EscalateCaseInput }>) {
  return Apollo.useMutation<{ escalateCase: Case }, { input: EscalateCaseInput }>(EscalateCaseDocument, options);
}

/* -- Ops -- */
export const QueryLogsDocument = gql`
  query QueryLogs($query: LogQueryInput!) {
    queryLogs(query: $query) {
      entries { id service severity message traceId spanId tenantId environment timestamp }
      nextPageToken
    }
  }
`;
export function useQueryLogsQuery(options: Apollo.QueryHookOptions<{ queryLogs: LogConnection }, { query: LogQueryInput }>) {
  return Apollo.useQuery<{ queryLogs: LogConnection }, { query: LogQueryInput }>(QueryLogsDocument, options);
}

export const QueryTracesDocument = gql`
  query QueryTraces($traceId: String, $service: String, $fromTime: Time, $toTime: Time, $limit: Int) {
    queryTraces(traceId: $traceId, service: $service, fromTime: $fromTime, toTime: $toTime, limit: $limit) {
      traceId spanId parentSpanId service operation durationMs status startTime
    }
  }
`;
export function useQueryTracesQuery(options?: Apollo.QueryHookOptions<{ queryTraces: TraceSpan[] }, { traceId?: string; service?: string; fromTime?: string; toTime?: string; limit?: number }>) {
  return Apollo.useQuery<{ queryTraces: TraceSpan[] }, { traceId?: string; service?: string; fromTime?: string; toTime?: string; limit?: number }>(QueryTracesDocument, options);
}

export const SloStatusDocument = gql`
  query SloStatus($service: String!, $window: String!) {
    sloStatus(service: $service, window: $window) {
      service errorRate p50LatencyMs p95LatencyMs p99LatencyMs availability window
    }
  }
`;
export function useSloStatusQuery(options: Apollo.QueryHookOptions<{ sloStatus: SLOStatus }, { service: string; window: string }>) {
  return Apollo.useQuery<{ sloStatus: SLOStatus }, { service: string; window: string }>(SloStatusDocument, options);
}

export const AlertsDocument = gql`
  query Alerts($service: String, $severity: String, $activeOnly: Boolean, $limit: Int) {
    alerts(service: $service, severity: $severity, activeOnly: $activeOnly, limit: $limit) {
      name service severity state summary firedAt
    }
  }
`;
export function useAlertsQuery(options?: Apollo.QueryHookOptions<{ alerts: Alert[] }, { service?: string; severity?: string; activeOnly?: boolean; limit?: number }>) {
  return Apollo.useQuery<{ alerts: Alert[] }, { service?: string; severity?: string; activeOnly?: boolean; limit?: number }>(AlertsDocument, options);
}

/* -- Admin -- */
export const TenantConfigDocument = gql`
  query TenantConfig($tenantId: ID!) {
    tenantConfig(tenantId: $tenantId) {
      id name status createdAt
      config { ruleSetId workflowTemplateId featureFlags { key enabled rolloutPercentage } version }
    }
  }
`;
export function useTenantConfigQuery(options: Apollo.QueryHookOptions<{ tenantConfig: Tenant | null }, { tenantId: string }>) {
  return Apollo.useQuery<{ tenantConfig: Tenant | null }, { tenantId: string }>(TenantConfigDocument, options);
}

export const FeatureFlagsDocument = gql`
  query FeatureFlags($tenantId: ID!) {
    featureFlags(tenantId: $tenantId) { key enabled rolloutPercentage }
  }
`;
export function useFeatureFlagsQuery(options: Apollo.QueryHookOptions<{ featureFlags: FeatureFlag[] }, { tenantId: string }>) {
  return Apollo.useQuery<{ featureFlags: FeatureFlag[] }, { tenantId: string }>(FeatureFlagsDocument, options);
}

export const CreateTenantDocument = gql`
  mutation CreateTenant($input: CreateTenantInput!) {
    createTenant(input: $input) { id name status createdAt }
  }
`;
export function useCreateTenantMutation(options?: Apollo.MutationHookOptions<{ createTenant: Tenant }, { input: CreateTenantInput }>) {
  return Apollo.useMutation<{ createTenant: Tenant }, { input: CreateTenantInput }>(CreateTenantDocument, options);
}

export const UpdateTenantRuleConfigDocument = gql`
  mutation UpdateTenantRuleConfig($input: UpdateTenantRuleConfigInput!) {
    updateTenantRuleConfig(input: $input) { id config { ruleSetId version } }
  }
`;
export function useUpdateTenantRuleConfigMutation(options?: Apollo.MutationHookOptions<{ updateTenantRuleConfig: Tenant }, { input: UpdateTenantRuleConfigInput }>) {
  return Apollo.useMutation<{ updateTenantRuleConfig: Tenant }, { input: UpdateTenantRuleConfigInput }>(UpdateTenantRuleConfigDocument, options);
}

export const UpdateTenantWorkflowConfigDocument = gql`
  mutation UpdateTenantWorkflowConfig($input: UpdateTenantWorkflowConfigInput!) {
    updateTenantWorkflowConfig(input: $input) { id config { workflowTemplateId version } }
  }
`;
export function useUpdateTenantWorkflowConfigMutation(options?: Apollo.MutationHookOptions<{ updateTenantWorkflowConfig: Tenant }, { input: UpdateTenantWorkflowConfigInput }>) {
  return Apollo.useMutation<{ updateTenantWorkflowConfig: Tenant }, { input: UpdateTenantWorkflowConfigInput }>(UpdateTenantWorkflowConfigDocument, options);
}

export const RulesDocument = gql`
  query Rules($tenantId: ID!, $includeDisabled: Boolean) {
    rules(tenantId: $tenantId, includeDisabled: $includeDisabled) {
      id tenantId name version action priority enabled createdAt updatedAt
    }
  }
`;
export function useRulesQuery(options: Apollo.QueryHookOptions<{ rules: Rule[] }, { tenantId: string; includeDisabled?: boolean }>) {
  return Apollo.useQuery<{ rules: Rule[] }, { tenantId: string; includeDisabled?: boolean }>(RulesDocument, options);
}

export const CreateRuleDocument = gql`
  mutation CreateRule($input: CreateRuleInput!) {
    createRule(input: $input) { id name version action priority enabled createdAt }
  }
`;
export function useCreateRuleMutation(options?: Apollo.MutationHookOptions<{ createRule: Rule }, { input: CreateRuleInput }>) {
  return Apollo.useMutation<{ createRule: Rule }, { input: CreateRuleInput }>(CreateRuleDocument, options);
}

export const UpdateRuleDocument = gql`
  mutation UpdateRule($input: UpdateRuleInput!) {
    updateRule(input: $input) { id name version action priority enabled updatedAt }
  }
`;
export function useUpdateRuleMutation(options?: Apollo.MutationHookOptions<{ updateRule: Rule }, { input: UpdateRuleInput }>) {
  return Apollo.useMutation<{ updateRule: Rule }, { input: UpdateRuleInput }>(UpdateRuleDocument, options);
}

export const DeleteRuleDocument = gql`
  mutation DeleteRule($input: DeleteRuleInput!) {
    deleteRule(input: $input)
  }
`;
export function useDeleteRuleMutation(options?: Apollo.MutationHookOptions<{ deleteRule: boolean }, { input: DeleteRuleInput }>) {
  return Apollo.useMutation<{ deleteRule: boolean }, { input: DeleteRuleInput }>(DeleteRuleDocument, options);
}

export const SimulateRuleDocument = gql`
  mutation SimulateRule($input: SimulateRuleInput!) {
    simulateRule(input: $input) { ruleId ruleName matched action reason }
  }
`;
export function useSimulateRuleMutation(options?: Apollo.MutationHookOptions<{ simulateRule: RuleEvalResult }, { input: SimulateRuleInput }>) {
  return Apollo.useMutation<{ simulateRule: RuleEvalResult }, { input: SimulateRuleInput }>(SimulateRuleDocument, options);
}

export const WorkflowTemplatesDocument = gql`
  query WorkflowTemplates($tenantId: ID!) {
    workflowTemplates(tenantId: $tenantId) {
      id tenantId name version states
      transitions { fromState toState requiredRole guards { type role condition } }
      createdAt updatedAt
    }
  }
`;
export function useWorkflowTemplatesQuery(options: Apollo.QueryHookOptions<{ workflowTemplates: WorkflowTemplate[] }, { tenantId: string }>) {
  return Apollo.useQuery<{ workflowTemplates: WorkflowTemplate[] }, { tenantId: string }>(WorkflowTemplatesDocument, options);
}

export const WorkflowTemplateDocument = gql`
  query WorkflowTemplate($templateId: ID!, $tenantId: ID!) {
    workflowTemplate(templateId: $templateId, tenantId: $tenantId) {
      id tenantId name version states
      transitions { fromState toState requiredRole guards { type role condition } }
      createdAt updatedAt
    }
  }
`;
export function useWorkflowTemplateQuery(options: Apollo.QueryHookOptions<{ workflowTemplate: WorkflowTemplate | null }, { templateId: string; tenantId: string }>) {
  return Apollo.useQuery<{ workflowTemplate: WorkflowTemplate | null }, { templateId: string; tenantId: string }>(WorkflowTemplateDocument, options);
}

export const CreateWorkflowTemplateDocument = gql`
  mutation CreateWorkflowTemplate($input: CreateWorkflowTemplateInput!) {
    createWorkflowTemplate(input: $input) { id name version states createdAt }
  }
`;
export function useCreateWorkflowTemplateMutation(options?: Apollo.MutationHookOptions<{ createWorkflowTemplate: WorkflowTemplate }, { input: CreateWorkflowTemplateInput }>) {
  return Apollo.useMutation<{ createWorkflowTemplate: WorkflowTemplate }, { input: CreateWorkflowTemplateInput }>(CreateWorkflowTemplateDocument, options);
}

export const UpdateWorkflowTemplateDocument = gql`
  mutation UpdateWorkflowTemplate($input: UpdateWorkflowTemplateInput!) {
    updateWorkflowTemplate(input: $input) { id name version states updatedAt }
  }
`;
export function useUpdateWorkflowTemplateMutation(options?: Apollo.MutationHookOptions<{ updateWorkflowTemplate: WorkflowTemplate }, { input: UpdateWorkflowTemplateInput }>) {
  return Apollo.useMutation<{ updateWorkflowTemplate: WorkflowTemplate }, { input: UpdateWorkflowTemplateInput }>(UpdateWorkflowTemplateDocument, options);
}

export const EvaluateTransitionDocument = gql`
  mutation EvaluateTransition($input: EvaluateTransitionInput!) {
    evaluateTransition(input: $input) { allowed reason }
  }
`;
export function useEvaluateTransitionMutation(options?: Apollo.MutationHookOptions<{ evaluateTransition: TransitionResult }, { input: EvaluateTransitionInput }>) {
  return Apollo.useMutation<{ evaluateTransition: TransitionResult }, { input: EvaluateTransitionInput }>(EvaluateTransitionDocument, options);
}

export const AuditTrailDocument = gql`
  query AuditTrail($query: AuditQueryInput!) {
    auditTrail(query: $query) {
      events { id tenantId actorId action resourceType resourceId sourceTopic hash previousHash occurredAt }
      nextPageToken
    }
  }
`;
export function useAuditTrailQuery(options: Apollo.QueryHookOptions<{ auditTrail: AuditConnection }, { query: AuditQueryInput }>) {
  return Apollo.useQuery<{ auditTrail: AuditConnection }, { query: AuditQueryInput }>(AuditTrailDocument, options);
}

export const VerifyChainIntegrityDocument = gql`
  query VerifyChainIntegrity($tenantId: ID!, $limit: Int) {
    verifyChainIntegrity(tenantId: $tenantId, limit: $limit) { valid eventsChecked brokenAtId }
  }
`;
export function useVerifyChainIntegrityQuery(options: Apollo.QueryHookOptions<{ verifyChainIntegrity: ChainVerification }, { tenantId: string; limit?: number }>) {
  return Apollo.useQuery<{ verifyChainIntegrity: ChainVerification }, { tenantId: string; limit?: number }>(VerifyChainIntegrityDocument, options);
}

export const AppendAuditEventDocument = gql`
  mutation AppendAuditEvent($tenantId: ID!, $actorId: ID!, $action: String!, $resourceType: String!, $resourceId: ID!) {
    appendAuditEvent(tenantId: $tenantId, actorId: $actorId, action: $action, resourceType: $resourceType, resourceId: $resourceId) {
      id action resourceType resourceId occurredAt
    }
  }
`;
export function useAppendAuditEventMutation(options?: Apollo.MutationHookOptions<{ appendAuditEvent: AuditEvent }, { tenantId: string; actorId: string; action: string; resourceType: string; resourceId: string }>) {
  return Apollo.useMutation<{ appendAuditEvent: AuditEvent }, { tenantId: string; actorId: string; action: string; resourceType: string; resourceId: string }>(AppendAuditEventDocument, options);
}

export const NotificationStatusDocument = gql`
  query NotificationStatus($notificationId: ID!, $tenantId: ID!) {
    notificationStatus(notificationId: $notificationId, tenantId: $tenantId) {
      id tenantId type recipient channel status attempts createdAt
    }
  }
`;
export function useNotificationStatusQuery(options: Apollo.QueryHookOptions<{ notificationStatus: Notification | null }, { notificationId: string; tenantId: string }>) {
  return Apollo.useQuery<{ notificationStatus: Notification | null }, { notificationId: string; tenantId: string }>(NotificationStatusDocument, options);
}

export const SendNotificationDocument = gql`
  mutation SendNotification($input: SendNotificationInput!) {
    sendNotification(input: $input) { notificationId status }
  }
`;
export function useSendNotificationMutation(options?: Apollo.MutationHookOptions<{ sendNotification: NotificationSendResult }, { input: SendNotificationInput }>) {
  return Apollo.useMutation<{ sendNotification: NotificationSendResult }, { input: SendNotificationInput }>(SendNotificationDocument, options);
}

export const UpdateNotificationPreferencesDocument = gql`
  mutation UpdateNotificationPreferences($input: UpdateNotificationPreferencesInput!) {
    updateNotificationPreferences(input: $input) { tenantId channel eventType enabled config }
  }
`;
export function useUpdateNotificationPreferencesMutation(options?: Apollo.MutationHookOptions<{ updateNotificationPreferences: NotificationPreference }, { input: UpdateNotificationPreferencesInput }>) {
  return Apollo.useMutation<{ updateNotificationPreferences: NotificationPreference }, { input: UpdateNotificationPreferencesInput }>(UpdateNotificationPreferencesDocument, options);
}
