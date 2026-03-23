export interface ApiError {
  error: {
    code: string;
    message: string;
    messageKey: string;
    details?: Record<string, unknown>;
    requestId: string;
  };
}

export interface PaginatedResponse<T> {
  data: T[];
  pagination: {
    page: number;
    pageSize: number;
    total: number;
    totalPages: number;
  };
}

export interface PaginationParams {
  page?: number;
  pageSize?: number;
}

export type ValueType = 'boolean' | 'string' | 'number' | 'json';
export type InputValueType = 'string' | 'number' | 'boolean';
export type FeatureAccessPolicy = 'public' | 'optional' | 'required';
export type AuthProfileType = 'api_key' | 'oidc_standard' | 'custom';
export type ExternalApiParamType = 'string' | 'number' | 'bool' | 'any';
export type ExternalApiParamLocation = 'url' | 'header' | 'body';
export type ExternalApiURLParamKind = 'domain' | 'path' | 'query';
export type ExternalApiValidationMode = 'httpCode' | 'responseBody' | 'both';
export type ExternalApiHTTPValidationMode = 'any_2xx' | 'status_codes';

export type MemberRole = 'owner' | 'admin' | 'editor' | 'viewer';

export interface SecurityPolicyList {
  managed: string[];
  inherited: string[];
  effective: string[];
}

export interface SecurityPolicy {
  corsOrigins: SecurityPolicyList;
  updatedAt?: string;
  updatedBy: string;
}

export interface UpdateSecurityPolicyRequest {
  corsOrigins: string[];
}

export interface Member {
  id: string;
  email: string;
  role: MemberRole;
  displayName: string;
  addedBy: string;
  createdAt: string;
  updatedAt: string;
}

export interface CreateMemberRequest {
  email: string;
  role: MemberRole;
}

export interface UpdateMemberRoleRequest {
  role: MemberRole;
}

export interface Feature {
  id: string;
  key: string;
  name: string;
  description: string;
  enabled: boolean;
  valueType: ValueType;
  defaultValue: unknown;
  metadata: Record<string, unknown>;
  tags: Tag[];
  activeFrom?: string | null;
  activeUntil?: string | null;
  environments?: string[];
  accessPolicy?: FeatureAccessPolicy;
  authProfileKey?: string;
  inputContract: InputContract;
  trialUntil?: string | null;
  trialValue?: unknown;
  tiers?: TierRef[];
  packs?: PackRef[];
  ruleCount?: number;
  rules?: Rule[];
  createdAt: string;
  updatedAt: string;
  createdBy: string;
  updatedBy: string;
}

export interface FeatureSummary {
  id: string;
  key: string;
  name: string;
  description: string;
  enabled: boolean;
  valueType: ValueType;
  environments?: string[];
  accessPolicy?: FeatureAccessPolicy;
  authProfileKey?: string;
  tags: Tag[];
  trialUntil?: string | null;
  tiers?: TierRef[];
  packCount: number;
  ruleCount: number;
  createdAt: string;
  updatedAt: string;
  createdBy: string;
  updatedBy: string;
}

export type FeatureListItem = Feature | FeatureSummary;

export interface ToggleFeatureResponse {
  message: string;
  enabled: boolean;
}

export interface ExternalApiBinding {
  externalApiKey: string;
  paramMappings: ParamMapping[];
  failMode: 'open' | 'closed';
  cacheTTL: number;
}

export interface ParamMapping {
  paramName: string;
  mode: 'input' | 'literal';
  inputPath?: string;
  literalValue?: string;
}

export interface Rule {
  id: string;
  name: string;
  priority: number;
  enabled: boolean;
  expression: string;
  value: unknown;
  sourceBindings: SourceBindings;
  externalApiBindings: ExternalApiBinding[];
  metadata: Record<string, unknown>;
  rolloutPercentage?: number | null;
  createdAt: string;
  updatedAt: string;
}

export interface InputContract {
  headers: InputHeader[];
  requestBodyExample?: Record<string, unknown>;
  requestBodySchema?: Record<string, unknown>;
}

export interface InputHeader {
  headerName: string;
  expressionKey: string;
  label: string;
  type: InputValueType;
  required: boolean;
  description?: string;
}

export interface SegmentSourceBinding {
  segmentKey: string;
  lookupPath: string;
}

export interface SourceBindings {
  segments: SegmentSourceBinding[];
}

export interface AuthProfile {
  id: string;
  key: string;
  name: string;
  active: boolean;
  type: AuthProfileType;
  config: Record<string, unknown>;
  cacheTTLSeconds?: number;
  version: number;
  hasSecret: boolean;
  createdAt: string;
  updatedAt: string;
  createdBy: string;
  updatedBy: string;
}

export interface ExternalApiHeaderTemplate {
  keyTemplate: string;
  valueTemplate: string;
}

export interface ExternalApiRequestConfig {
  method: 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE';
  urlTemplate: string;
  headers: ExternalApiHeaderTemplate[];
  bodyTemplate?: unknown | null;
}

export interface ExternalApiParam {
  name: string;
  type: ExternalApiParamType;
  required: boolean;
  locations: ExternalApiParamLocation[];
  urlKind?: ExternalApiURLParamKind;
}

export interface ExternalApiExpressionVariable {
  name: string;
  type: ExternalApiParamType;
  required: boolean;
}

export interface ExternalApiHTTPValidation {
  mode: ExternalApiHTTPValidationMode;
  codes?: number[];
}

export interface ExternalApiBodyValidation {
  expression: string;
  schema?: Record<string, unknown>;
  sampleResponseText?: string;
}

export interface ExternalApiResponseValidation {
  mode: ExternalApiValidationMode;
  http: ExternalApiHTTPValidation;
  body: ExternalApiBodyValidation;
}

export interface ExternalApi {
  id: string;
  key: string;
  name: string;
  active: boolean;
  request: ExternalApiRequestConfig;
  params: ExternalApiParam[];
  expressionVariables?: ExternalApiExpressionVariable[];
  responseValidation: ExternalApiResponseValidation;
  hasSecrets: boolean;
  version: number;
  createdAt: string;
  updatedAt: string;
  createdBy: string;
  updatedBy: string;
}

export interface ExternalApiTestRequestDetails {
  url?: string;
  method?: string;
  headers?: Record<string, string>;
  body?: unknown;
}

export interface ExternalApiTestEvaluationFinal {
  mode: ExternalApiValidationMode;
  passed: boolean;
}

export interface ExternalApiTestEvaluationHTTP {
  applied: boolean;
  passed: boolean;
  mode: ExternalApiHTTPValidationMode;
  expectedCodes: number[];
  actualStatus: number;
}

export interface ExternalApiTestEvaluationExpression {
  applied: boolean;
  passed: boolean;
  expression: string;
  resolvedExpression?: string | null;
  error: string | null;
}

export interface ExternalApiTestEvaluations {
  final: ExternalApiTestEvaluationFinal;
  http: ExternalApiTestEvaluationHTTP;
  expression: ExternalApiTestEvaluationExpression;
}

export interface ExternalApiTestDetails {
  request?: ExternalApiTestRequestDetails;
  responseText?: string;
  responseHeaders?: Record<string, string>;
  responseBody?: unknown;
  evaluations?: ExternalApiTestEvaluations;
}

export interface ExternalApiTestResponse {
  ok: boolean;
  attempted: boolean;
  httpStatus?: number;
  details?: ExternalApiTestDetails;
}

export interface ExternalApiExpressionSymbol {
  path: string;
  type: string;
  description?: string;
}

export interface ExternalApiExpressionAction {
  id: string;
  label: string;
  detail?: string;
  category: string;
  appliesTo: string[];
  template: string;
  priority: number;
}

export interface ExternalApiExpressionProfile {
  keywords: string[];
  symbols: ExternalApiExpressionSymbol[];
  actions: ExternalApiExpressionAction[];
}

export interface Segment {
  id: string;
  key: string;
  name: string;
  description: string;
  metadata: Record<string, unknown>;
  recordCount: number;
  recordKeyPath?: string;
  previewFields: string[];
  sourceType?: 'csv' | 'json';
  lastImportAt: string | null;
  createdAt: string;
  updatedAt: string;
  createdBy: string;
  updatedBy: string;
}

export interface SegmentSchema {
  segmentKey: string;
  schema: Record<string, unknown>;
  recordKeyPath: string;
  activeDatasetVersion?: string;
  previewFields: string[];
  sourceType?: 'csv' | 'json';
  lastImportAt: string | null;
  recordCount: number;
}

export interface SegmentRecord {
  id: string;
  recordKey: string;
  attributes: Record<string, unknown>;
  createdAt: string;
}

export interface AuditError {
  id: string;
  featureKey: string;
  ruleId: string;
  errorType: string;
  message: string;
  tenantId: string;
  campusId: string;
  programId: string;
  requestId: string;
  createdAt: string;
}

export interface ExpressionValidateResponse {
  valid: boolean;
  error?: string;
}

export interface ExpressionTestResponse {
  result: unknown;
  matched: boolean;
  error?: string;
}

export interface ExpressionSchemaField {
  name: string;
  type: string;
  description: string;
  category: string;
}

export interface ExpressionSchemaFunction {
  name: string;
  signature: string;
  description: string;
}

export interface ExpressionSchema {
  fields: ExpressionSchemaField[];
  functions: ExpressionSchemaFunction[];
}

export interface FeatureExpressionField {
  path: string;
  label: string;
  description?: string;
  type: string;
  example?: unknown;
  group: string;
}

export interface FeatureExpressionSchema {
  headers: FeatureExpressionField[];
  requestBody: FeatureExpressionField[];
  derived: FeatureExpressionField[];
  advancedMode: boolean;
}

export interface FeatureExpressionScenario {
  headers: Record<string, string>;
  requestBody: Record<string, unknown>;
}

export interface ResolvedSegmentSource {
  segmentKey: string;
  lookupPath: string;
  lookupValue?: unknown;
  found: boolean;
  data?: Record<string, unknown>;
}

export interface FeatureExpressionTestResponse {
  result: unknown;
  matched: boolean;
  derived?: Record<string, unknown>;
  resolvedSources?: ResolvedSegmentSource[];
  explanation?: string;
  error?: string;
}

export interface Tag {
  key: string;
  name: string;
  color: string;
}

export interface TierRef {
  key: string;
  name: string;
  color: string;
}

export type ApiKeyType = 'admin';

export type ApiKeyPermission =
  | 'features.read'
  | 'features.write'
  | 'segments.read'
  | 'segments.write'
  | 'packs.read'
  | 'packs.write'
  | 'audit.read';

export const AllowedAPIKeyPermissions: ApiKeyPermission[] = [
  'features.read',
  'features.write',
  'segments.read',
  'segments.write',
  'packs.read',
  'packs.write',
  'audit.read',
];

export interface ApiKey {
  id: string;
  name: string;
  description: string;
  type: ApiKeyType;
  permissions: ApiKeyPermission[];
  prefix: string;
  createdBy: string;
  createdAt: string;
  expiresAt: string | null;
  lastUsedAt: string | null;
  revoked: boolean;
}

export interface CreateApiKeyRequest {
  name: string;
  description?: string;
  type: ApiKeyType;
  permissions?: ApiKeyPermission[];
  expiresAt?: string | null;
}

export interface CreateApiKeyResponse {
  key: string;
  id: string;
  name: string;
  prefix: string;
  type: ApiKeyType;
  description: string;
  permissions: ApiKeyPermission[];
  createdBy: string;
  createdAt: string;
  expiresAt: string | null;
}

export interface RotateApiKeyResponse {
  key: string;
  id: string;
  name: string;
  prefix: string;
  type: ApiKeyType;
  description: string;
  permissions: ApiKeyPermission[];
  createdBy: string;
  createdAt: string;
  expiresAt: string | null;
}

export type TargetType = 'tenant' | 'campus' | 'program';

export interface PackRef {
  key: string;
  name: string;
}

export interface Pack {
  id: string;
  key: string;
  name: string;
  description: string;
  featureKeys: string[];
  enabled: boolean;
  metadata: Record<string, unknown>;
  tierKey?: string | null;
  tier?: TierRef | null;
  inheritsFrom?: string[];
  trialUntil?: string | null;
  resolvedFeatureCount?: number;
  createdAt: string;
  updatedAt: string;
  createdBy: string;
  updatedBy: string;
}

export interface PackActivation {
  id: string;
  packKey: string;
  targetType: TargetType;
  targetId: string;
  activatedAt: string;
  activatedBy: string;
  expiresAt?: string | null;
  metadata?: Record<string, unknown>;
}

export interface CreatePackRequest {
  key: string;
  name: string;
  description?: string;
  featureKeys?: string[];
  tierKey?: string | null;
  inheritsFrom?: string[];
  trialUntil?: string | null;
}

export interface UpdatePackRequest {
  name?: string;
  description?: string;
  featureKeys?: string[];
  tierKey?: string | null;
  inheritsFrom?: string[];
  trialUntil?: string | null;
}

export interface ActivatePackRequest {
  targetType: TargetType;
  targetId: string;
  expiresAt?: string | null;
  metadata?: Record<string, unknown>;
}

export interface DeactivatePackRequest {
  targetType: TargetType;
  targetId: string;
}

export type ExperimentStatus = 'draft' | 'running' | 'paused' | 'completed';

export interface Variant {
  key: string;
  value: unknown;
  weight: number;
}

export interface ExperimentMetric {
  key: string;
  name: string;
  description?: string;
}

export interface Experiment {
  id: string;
  workspaceKey: string;
  featureKey: string;
  name: string;
  description: string;
  status: ExperimentStatus;
  variants: Variant[];
  metrics: ExperimentMetric[];
  winnerKey?: string;
  startedAt?: string | null;
  completedAt?: string | null;
  createdBy: string;
  createdAt: string;
  updatedAt: string;
}

export interface VariantStats {
  variantKey: string;
  exposures: number;
  conversions: number;
  conversionRate: number;
  confidenceLow: number;
  confidenceHigh: number;
}

export interface ExperimentResults {
  experimentId: string;
  totalExposures: number;
  totalConversions: number;
  variants: VariantStats[];
  isSignificant: boolean;
}

export interface CreateExperimentRequest {
  featureKey: string;
  name: string;
  description?: string;
  variants: Variant[];
  metrics?: ExperimentMetric[];
}

export interface UpdateExperimentRequest {
  name: string;
  description?: string;
  variants: Variant[];
  metrics?: ExperimentMetric[];
}
