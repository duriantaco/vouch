package vouch

type Risk string

const (
	SpecSchemaVersion     = "vouch.spec.v0"
	ManifestSchemaVersion = "vouch.manifest.v0"
	EvidenceSchemaVersion = "vouch.evidence.v0"
	ASTSchemaVersion      = "vouch.ast.v0"
	IRSchemaVersion       = "vouch.ir.v0"
	PlanSchemaVersion     = "vouch.plan.v0"
	ConfigSchemaVersion   = "vouch.config.v0"
	TestMapSchemaVersion  = "vouch.test_map.v0"
)

const (
	RiskLow      Risk = "low"
	RiskMedium   Risk = "medium"
	RiskHigh     Risk = "high"
	RiskCritical Risk = "critical"
)

var riskRank = map[Risk]int{
	RiskLow:      0,
	RiskMedium:   1,
	RiskHigh:     2,
	RiskCritical: 3,
}

type Spec struct {
	Version    string       `json:"version"`
	ID         string       `json:"id"`
	Owner      string       `json:"owner"`
	OwnedPaths []string     `json:"owned_paths,omitempty"`
	Risk       Risk         `json:"risk"`
	Behavior   []string     `json:"behavior"`
	Security   []string     `json:"security"`
	Tests      SpecTests    `json:"tests"`
	Runtime    SpecRuntime  `json:"runtime"`
	Rollback   SpecRollback `json:"rollback"`
}

type SpecTests struct {
	Required []string `json:"required"`
}

type SpecRuntime struct {
	Metrics []string `json:"metrics"`
	Alerts  []string `json:"alerts"`
}

type SpecRollback struct {
	Strategy string `json:"strategy"`
	Flag     string `json:"flag"`
}

type ObligationKind string

const (
	ObligationBehavior      ObligationKind = "behavior"
	ObligationSecurity      ObligationKind = "security"
	ObligationRequiredTest  ObligationKind = "required_test"
	ObligationRuntimeSignal ObligationKind = "runtime_signal"
	ObligationRollback      ObligationKind = "rollback"
)

type EvidenceKind string

const (
	EvidenceBehaviorTrace EvidenceKind = "behavior_trace"
	EvidenceSecurityCheck EvidenceKind = "security_check"
	EvidenceTestCoverage  EvidenceKind = "test_coverage"
	EvidenceRuntimeMetric EvidenceKind = "runtime_metric"
	EvidenceRollbackPlan  EvidenceKind = "rollback_plan"
)

type Intent struct {
	Feature        string
	Owner          string
	OwnedPaths     []string
	Risk           Risk
	Goal           string
	Behavior       []string
	Security       []string
	RequiredTests  []string
	RuntimeMetrics []string
	RuntimeAlerts  []string
	Rollback       SpecRollback
}

type SourceSpan struct {
	File   string `json:"file"`
	Line   int    `json:"line"`
	Column int    `json:"column"`
}

type Diagnostic struct {
	Severity string     `json:"severity"`
	Code     string     `json:"code"`
	Message  string     `json:"message"`
	Path     string     `json:"path,omitempty"`
	Span     SourceSpan `json:"span"`
}

type IntentAST struct {
	Version     string       `json:"version"`
	File        string       `json:"file"`
	Nodes       []ASTNode    `json:"nodes"`
	Diagnostics []Diagnostic `json:"diagnostics"`
}

type ASTNode struct {
	Kind     string     `json:"kind"`
	Key      string     `json:"key"`
	Value    string     `json:"value,omitempty"`
	Values   []ASTValue `json:"values,omitempty"`
	Children []ASTNode  `json:"children,omitempty"`
	Span     SourceSpan `json:"span"`
}

type ASTValue struct {
	Value string     `json:"value"`
	Span  SourceSpan `json:"span"`
}

type IR struct {
	Version        string       `json:"version"`
	Feature        string       `json:"feature"`
	Owner          string       `json:"owner"`
	Risk           Risk         `json:"risk"`
	Obligations    []Obligation `json:"obligations"`
	RequiredChecks []string     `json:"required_checks"`
	RuntimeSignals []string     `json:"runtime_signals"`
	Rollback       SpecRollback `json:"rollback"`
	ReleasePolicy  []string     `json:"release_policy"`
}

type VerificationPlan struct {
	Version        string       `json:"version"`
	Feature        string       `json:"feature"`
	Risk           Risk         `json:"risk"`
	SpecsTouched   []string     `json:"specs_touched"`
	Obligations    []Obligation `json:"obligations"`
	RequiredChecks []string     `json:"required_checks"`
	VerifierRoles  []string     `json:"verifier_roles"`
	RuntimeSignals []string     `json:"runtime_signals"`
	Rollback       SpecRollback `json:"rollback"`
	ReleasePolicy  []string     `json:"release_policy"`
	Diagnostics    []Diagnostic `json:"diagnostics"`
}

type VerifierPacket struct {
	Verifier       string       `json:"verifier"`
	Focus          string       `json:"focus"`
	Obligations    []Obligation `json:"obligations"`
	RequiredOutput string       `json:"required_output"`
}

type TestObligationsArtifact struct {
	Version     string       `json:"version"`
	Feature     string       `json:"feature"`
	Obligations []Obligation `json:"obligations"`
}

type ReleasePolicyArtifact struct {
	Version        string   `json:"version"`
	Feature        string   `json:"feature"`
	Risk           Risk     `json:"risk"`
	ReleasePolicy  []string `json:"release_policy"`
	RequiredChecks []string `json:"required_checks"`
}

type Obligation struct {
	ID               string         `json:"id"`
	Kind             ObligationKind `json:"kind"`
	Text             string         `json:"text"`
	Risk             Risk           `json:"risk"`
	Severity         string         `json:"severity"`
	Source           string         `json:"source"`
	RequiredEvidence EvidenceKind   `json:"required_evidence"`
}

type Manifest struct {
	Version       string           `json:"version"`
	Task          Task             `json:"task"`
	Change        Change           `json:"change"`
	Agent         Agent            `json:"agent"`
	Verification  Verification     `json:"verification"`
	Runtime       ManifestRuntime  `json:"runtime"`
	Rollback      ManifestRollback `json:"rollback"`
	Uncertainties []string         `json:"uncertainties"`
}

type Task struct {
	ID      string `json:"id"`
	Summary string `json:"summary"`
}

type Change struct {
	Risk             Risk     `json:"risk"`
	SpecsTouched     []string `json:"specs_touched"`
	BehaviorChanged  bool     `json:"behavior_changed"`
	MigrationChanged bool     `json:"migration_changed"`
	ExternalEffects  []string `json:"external_effects"`
	ChangedFiles     []string `json:"changed_files"`
}

type Agent struct {
	Name  string `json:"name"`
	RunID string `json:"run_id"`
	Model string `json:"model"`
}

type Verification struct {
	CoversBehavior []string           `json:"covers_behavior"`
	Commands       []string           `json:"commands"`
	TestsAdded     []string           `json:"tests_added"`
	CoversTests    []string           `json:"covers_tests"`
	CoversSecurity []string           `json:"covers_security"`
	TestResults    TestResults        `json:"test_results"`
	Artifacts      []EvidenceArtifact `json:"artifacts"`
}

type EvidenceArtifact struct {
	ID               string       `json:"id"`
	Kind             EvidenceKind `json:"kind"`
	Producer         string       `json:"producer,omitempty"`
	Command          string       `json:"command,omitempty"`
	Path             string       `json:"path,omitempty"`
	SHA256           string       `json:"sha256,omitempty"`
	SignatureBundle  string       `json:"signature_bundle,omitempty"`
	SignerIdentity   string       `json:"signer_identity,omitempty"`
	SignerOIDCIssuer string       `json:"signer_oidc_issuer,omitempty"`
	ExitCode         *int         `json:"exit_code"`
	Obligations      []string     `json:"obligations"`
}

type TestResults struct {
	Passed int `json:"passed"`
	Failed int `json:"failed"`
}

type ManifestRuntime struct {
	Metrics []string `json:"metrics"`
	Canary  Canary   `json:"canary"`
}

type Canary struct {
	Enabled        bool `json:"enabled"`
	InitialPercent int  `json:"initial_percent"`
}

type ManifestRollback struct {
	Strategy     string   `json:"strategy"`
	Flag         string   `json:"flag"`
	Compensation []string `json:"compensation"`
}

type Finding struct {
	Verifier    string `json:"verifier"`
	Severity    string `json:"severity"`
	Decision    string `json:"decision"`
	Claim       string `json:"claim"`
	Evidence    string `json:"evidence"`
	RequiredFix string `json:"required_fix"`
}

func (f Finding) Blocks() bool {
	return f.Decision == "block"
}

type Evidence struct {
	Version                string                      `json:"version"`
	Repo                   string                      `json:"repo"`
	ManifestPath           string                      `json:"manifest_path"`
	Compilation            CompilationStats            `json:"compilation"`
	Manifest               Manifest                    `json:"manifest"`
	Specs                  map[string]Spec             `json:"specs"`
	IRs                    map[string]IR               `json:"irs"`
	VerificationPlans      map[string]VerificationPlan `json:"verification_plans"`
	Diagnostics            []Diagnostic                `json:"diagnostics"`
	SignedEvidenceRequired bool                        `json:"signed_evidence_required"`
	SpecErrors             []string                    `json:"spec_errors"`
	ManifestErrors         []string                    `json:"manifest_errors"`
	ArtifactResults        []ArtifactResult            `json:"artifact_results"`
	InvalidEvidence        []InvalidEvidence           `json:"invalid_evidence"`
	RequiredObligations    map[string][]Obligation     `json:"required_obligations"`
	CoveredObligations     map[string][]Obligation     `json:"covered_obligations"`
	MissingObligations     map[string][]Obligation     `json:"missing_obligations"`
	RequiredTests          map[string][]string         `json:"required_tests"`
	CoveredTests           map[string][]string         `json:"covered_tests"`
	MissingTests           map[string][]string         `json:"missing_tests"`
	RequiredSecurity       map[string][]string         `json:"required_security"`
	CoveredSecurity        map[string][]string         `json:"covered_security"`
	MissingSecurity        map[string][]string         `json:"missing_security"`
	Findings               []Finding                   `json:"findings"`
	Decision               string                      `json:"decision"`
	Reasons                []string                    `json:"reasons"`
}

type CompilationStats struct {
	SpecsLoaded      int `json:"specs_loaded"`
	SpecsCompiled    int `json:"specs_compiled"`
	SpecsSkipped     int `json:"specs_skipped"`
	ObligationsBuilt int `json:"obligations_built"`
}

type ArtifactResult struct {
	ID                 string       `json:"id"`
	Kind               EvidenceKind `json:"kind"`
	Path               string       `json:"path,omitempty"`
	ResolvedPath       string       `json:"resolved_path,omitempty"`
	Status             string       `json:"status"`
	HashVerified       bool         `json:"hash_verified"`
	SignatureVerified  bool         `json:"signature_verified"`
	CoveredObligations []string     `json:"covered_obligations"`
	FailedTests        []string     `json:"failed_tests,omitempty"`
	Issues             []string     `json:"issues,omitempty"`
}

type InvalidEvidence struct {
	Artifact string `json:"artifact"`
	Code     string `json:"code"`
	Message  string `json:"message"`
}

type GateResult struct {
	Version            string              `json:"version"`
	Decision           string              `json:"decision"`
	Reasons            []string            `json:"reasons"`
	Compilation        CompilationStats    `json:"compilation"`
	SpecErrors         []string            `json:"spec_errors"`
	ManifestErrors     []string            `json:"manifest_errors"`
	ArtifactResults    []ArtifactResult    `json:"artifact_results"`
	InvalidEvidence    []InvalidEvidence   `json:"invalid_evidence"`
	MissingObligations map[string][]string `json:"missing_obligations"`
	CoveredObligations map[string][]string `json:"covered_obligations"`
	PolicyRulesFired   []string            `json:"policy_rules_fired"`
	FiredPolicyRule    string              `json:"fired_policy_rule"`
}

type Config struct {
	Version      string   `json:"version"`
	Profiles     []string `json:"profiles"`
	Commands     []string `json:"commands"`
	ArtifactDir  string   `json:"artifact_dir"`
	ManifestDir  string   `json:"manifest_dir"`
	BuildDir     string   `json:"build_dir"`
	IgnoredPaths []string `json:"ignored_paths"`
}

type InitResult struct {
	Version     string   `json:"version"`
	Repo        string   `json:"repo"`
	ConfigPath  string   `json:"config_path"`
	Profiles    []string `json:"profiles"`
	Commands    []string `json:"commands"`
	CreatedDirs []string `json:"created_dirs"`
	Created     bool     `json:"created"`
}

type ContractSuggestion struct {
	Name       string   `json:"name"`
	Profile    string   `json:"profile"`
	OwnedPaths []string `json:"owned_paths"`
	TestPaths  []string `json:"test_paths"`
	Confidence string   `json:"confidence"`
}

type TestMap struct {
	Version  string              `json:"version"`
	Mappings map[string][]string `json:"mappings"`
}

type JUnitMapResult struct {
	Version            string   `json:"version"`
	InputPath          string   `json:"input_path"`
	OutputPath         string   `json:"output_path"`
	TestMapPath        string   `json:"test_map_path"`
	Cases              int      `json:"cases"`
	CoveredObligations []string `json:"covered_obligations"`
}
