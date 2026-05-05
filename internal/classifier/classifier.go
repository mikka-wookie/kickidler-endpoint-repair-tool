package classifier

import (
	"fmt"
	"path/filepath"
	"reflect"
	"strings"

	"kigrepair/internal/detector"
)

const (
	verificationPassed  = "passed"
	verificationWarning = "warning"
	verificationFailed  = "failed"
	verificationSkipped = "skipped"
)

var primaryPriority = []string{
	CodeHealthy,
	CodeNotInstalled,
	CodeServiceBinaryMissing,
	CodeWMIHiddenModeInconsistent,
	CodePartialMSILeftovers,
	CodePartialFilesLeftover,
	CodeDefenderExclusionMissing,
	CodeServiceNotRunning,
	CodeVerificationFailed,
	CodeVerificationWarning,
	CodeDefenderStatusUnavailable,
	CodeUnknownInstallState,
}

func Classify(input ClassificationInput) ClassificationResult {
	issues := make([]Issue, 0)
	add := func(issue *Issue) {
		if issue == nil || hasIssue(issues, issue.Code) {
			return
		}
		issues = append(issues, *issue)
	}

	add(detectHealthy(input.Detection, input.Verification))
	add(detectNotInstalled(input.Detection))
	add(detectServiceBinaryMissing(input.Detection))
	add(detectWMIHiddenModeInconsistent(input.Detection))
	add(detectPartialMSILeftovers(input.Detection))
	add(detectPartialFilesLeftover(input.Detection))
	add(detectDefenderExclusionMissing(input.Detection))
	add(detectServiceNotRunning(input.Detection, input.Verification))
	add(detectVerificationFailed(input.Verification))
	add(detectVerificationWarning(input.Verification))
	add(detectDefenderStatusUnavailable(input.Detection, input.Verification))
	add(detectUnknownInstallState(input.Detection))

	if len(issues) == 0 {
		add(unknownInstallStateIssue([]string{"No classification rule matched the available detection or verification data"}))
	}

	primary := choosePrimaryIssue(issues)
	result := ClassificationResult{
		Status:            classificationStatus(primary),
		PrimaryIssue:      primary,
		Issues:            issues,
		RecommendedAction: primaryAction(primary),
	}
	result.SupportSummary = SupportSummary(input, result)
	return result
}

func detectHealthy(d *detector.DetectionReport, v any) *Issue {
	if d == nil || d.Health != detector.GrabberHealthHealthy {
		return nil
	}
	if v != nil && verificationOverallStatus(v) != verificationPassed {
		return nil
	}
	if len(d.RequiredDefenderPaths) > 0 && (!d.Defender.Available || len(d.MissingDefenderPaths) > 0) {
		return nil
	}
	evidence := []string{"Detection health is healthy"}
	if service := primaryService(d); service != nil {
		evidence = append(evidence, fmt.Sprintf("Service %s status is %s", service.Name, valueOrDash(service.Status)))
	}
	if d.ServiceExecutablePath != "" && d.ServiceExecutableExists {
		evidence = append(evidence, "Service executable exists: "+d.ServiceExecutablePath)
	}
	if len(d.RequiredDefenderPaths) > 0 && len(d.MissingDefenderPaths) == 0 && d.Defender.Available {
		evidence = append(evidence, "Defender exclusion coverage is valid")
	}
	return issue(CodeHealthy, SeverityInfo, "Grabber installation is healthy", "Grabber satisfies detection and verification requirements.", evidence, "No repair required.")
}

func detectNotInstalled(d *detector.DetectionReport) *Issue {
	if d == nil || d.Health != detector.GrabberHealthNotInstalled {
		return nil
	}
	if anyServiceExists(d) || anyFileExists(d) || anyRegistryExists(d) {
		return nil
	}
	return issue(CodeNotInstalled, SeverityInfo, "Grabber is not installed", "No supported Grabber service, known install root, files, or registry presence was detected.", nil, "Install Grabber or run repair with installer and invite, depending on the support case.")
}

func detectServiceBinaryMissing(d *detector.DetectionReport) *Issue {
	if d == nil || primaryService(d) == nil || strings.TrimSpace(d.ServiceExecutablePath) == "" || d.ServiceExecutableExists {
		return nil
	}
	if d.InstallMode == detector.InstallModeHiddenWMI || pathLooksWMI(d.ServiceExecutablePath) {
		return nil
	}
	evidence := []string{
		"Primary service: " + d.PrimaryService,
		"Service executable path: " + d.ServiceExecutablePath,
		"Service executable exists: false",
	}
	return issue(CodeServiceBinaryMissing, SeverityCritical, "Service executable is missing", "A supported service exists and points to an executable path, but the executable is missing.", evidence, "Run repair with a valid installer and invite. Check Defender exclusion coverage before or during repair.")
}

func detectDefenderExclusionMissing(d *detector.DetectionReport) *Issue {
	if d == nil || strings.TrimSpace(d.InstallRoot) == "" || !d.Defender.Available || len(d.RequiredDefenderPaths) == 0 || len(d.MissingDefenderPaths) == 0 {
		return nil
	}
	evidence := append([]string{"Install root: " + d.InstallRoot}, prefixValues("Missing Defender exclusion: ", d.MissingDefenderPaths)...)
	return issue(CodeDefenderExclusionMissing, SeverityWarning, "Defender exclusion is missing", "The detected install root is not covered by the required Microsoft Defender exclusion.", evidence, "Run defender ensure or repair with admin rights to add the required Defender exclusion.")
}

func detectPartialMSILeftovers(d *detector.DetectionReport) *Issue {
	if d == nil || !anyMSIRegistryExists(d) || d.Health == detector.GrabberHealthHealthy {
		return nil
	}
	if anyServiceExists(d) && strings.TrimSpace(d.InstallRoot) != "" {
		return nil
	}
	return issue(CodePartialMSILeftovers, SeverityWarning, "MSI registry leftovers found", "MSI product registry keys exist, but no healthy service/install root was detected.", existingRegistryEvidence(d, true), "Run cleanup followed by install, or run full repair with installer and invite.")
}

func detectPartialFilesLeftover(d *detector.DetectionReport) *Issue {
	if d == nil || anyServiceExists(d) || !anyFileExists(d) {
		return nil
	}
	if d.Health != detector.GrabberHealthPartiallyRemoved && d.Health != detector.GrabberHealthBroken {
		return nil
	}
	return issue(CodePartialFilesLeftover, SeverityWarning, "Installation files remain without service", "Known Grabber files or directories exist, but no supported service exists.", existingFileEvidence(d), "Run cleanup dry-run first, review cleanup plan, then run cleanup or repair.")
}

func detectWMIHiddenModeInconsistent(d *detector.DetectionReport) *Issue {
	if d == nil {
		return nil
	}
	wmiIndicated := d.InstallMode == detector.InstallModeHiddenWMI || pathLooksWMI(d.InstallRoot) || pathLooksWMI(d.BinaryDir) || pathLooksWMI(d.ServiceExecutablePath) || wmiFilesExist(d)
	if !wmiIndicated {
		return nil
	}
	inconsistent := false
	evidence := []string{"Hidden WMI mode is indicated"}
	if d.InstallMode == detector.InstallModeHiddenWMI && primaryService(d) != nil && !d.ServiceExecutableExists {
		inconsistent = true
		evidence = append(evidence, "WMI service executable is missing: "+d.ServiceExecutablePath)
	}
	if wmiFilesExist(d) && primaryService(d) == nil {
		inconsistent = true
		evidence = append(evidence, "WMI files exist without a supported service")
	}
	if service := primaryService(d); service != nil && strings.EqualFold(service.Name, "WmiProviderSE") && !service.ExpectedImagePathMatch {
		inconsistent = true
		evidence = append(evidence, "WmiProviderSE ImagePath does not match expected WMI path")
	}
	if !inconsistent {
		return nil
	}
	return issue(CodeWMIHiddenModeInconsistent, SeverityCritical, "Hidden WMI mode is inconsistent", "Hidden WMI service, path, or file signals are inconsistent.", evidence, "Run repair. If repair cannot restore hidden WMI mode, collect support bundle for escalation.")
}

func detectServiceNotRunning(d *detector.DetectionReport, v any) *Issue {
	if d == nil || primaryService(d) == nil || !d.ServiceExecutableExists || serviceRunning(d) {
		return nil
	}
	if v != nil && !verificationCheckStatus(v, "primary_service_running", verificationFailed, verificationWarning) {
		return nil
	}
	evidence := []string{"Primary service: " + d.PrimaryService, "Service status: " + serviceStatus(d), "Service executable exists: true"}
	return issue(CodeServiceNotRunning, SeverityWarning, "Service is not running", "The primary supported service exists and its executable exists, but the service is stopped or not running.", evidence, "Run repair. If repair fails, inspect service start errors and MSI logs.")
}

func detectVerificationFailed(v any) *Issue {
	if v == nil || verificationOverallStatus(v) != verificationFailed {
		return nil
	}
	return issue(CodeVerificationFailed, SeverityCritical, "Verification failed", "Final verification failed and no more specific classifier issue may explain the failure.", verificationEvidence(v), "Review failed verification checks and run repair if installation should be present.")
}

func detectVerificationWarning(v any) *Issue {
	if v == nil || verificationOverallStatus(v) != verificationWarning {
		return nil
	}
	return issue(CodeVerificationWarning, SeverityWarning, "Verification completed with warnings", "Final verification produced warnings and no more specific classifier issue may explain them.", verificationEvidence(v), "Review warning checks. Repair may not be required unless user symptoms persist.")
}

func detectDefenderStatusUnavailable(d *detector.DetectionReport, v any) *Issue {
	if d != nil && len(d.RequiredDefenderPaths) > 0 && !d.Defender.Available {
		evidence := []string{"Defender availability: false"}
		if d.Defender.Error != "" {
			evidence = append(evidence, "Defender error: "+d.Defender.Error)
		}
		return issue(CodeDefenderStatusUnavailable, SeverityWarning, "Defender status is unavailable", "Defender status query failed, so required exclusion coverage could not be confirmed.", evidence, "Run as admin if needed, verify Microsoft Defender availability, or check exclusions manually.")
	}
	if v != nil && strings.EqualFold(verificationStringField(v, "DefenderStatus"), "unavailable") {
		return issue(CodeDefenderStatusUnavailable, SeverityWarning, "Defender status is unavailable", "Verification could not confirm Defender exclusion coverage.", []string{"Verification Defender status: unavailable"}, "Run as admin if needed, verify Microsoft Defender availability, or check exclusions manually.")
	}
	return nil
}

func detectUnknownInstallState(d *detector.DetectionReport) *Issue {
	if d == nil {
		return unknownInstallStateIssue([]string{"Detection result is unavailable"})
	}
	if d.Health == detector.GrabberHealthUnknown || d.InstallMode == detector.InstallModeUnknown {
		return unknownInstallStateIssue([]string{"Detection health: " + string(d.Health), "Install mode: " + string(d.InstallMode)})
	}
	return nil
}

func choosePrimaryIssue(issues []Issue) *Issue {
	for _, code := range primaryPriority {
		for i := range issues {
			if issues[i].Code == code {
				issue := issues[i]
				return &issue
			}
		}
	}
	if len(issues) == 0 {
		return nil
	}
	issue := issues[0]
	return &issue
}

func classificationStatus(primary *Issue) string {
	if primary == nil {
		return StatusInconclusive
	}
	switch primary.Code {
	case CodeHealthy:
		return StatusNoIssue
	case CodeUnknownInstallState:
		return StatusInconclusive
	default:
		return StatusIssueDetected
	}
}

func primaryAction(primary *Issue) string {
	if primary == nil {
		return "Review detection, verification, and support bundle."
	}
	return primary.Action
}

func issue(code, severity, title, description string, evidence []string, action string) *Issue {
	return &Issue{Code: code, Severity: severity, Title: title, Description: description, Evidence: compact(evidence), Action: action}
}

func unknownInstallStateIssue(evidence []string) *Issue {
	return issue(CodeUnknownInstallState, SeverityWarning, "Install state is unknown", "Detection returned conflicting or insufficient signals, so the installation state cannot be classified confidently.", evidence, "Review detection, verification, and support bundle. Escalate if classification remains inconclusive.")
}

func hasIssue(issues []Issue, code string) bool {
	for _, issue := range issues {
		if issue.Code == code {
			return true
		}
	}
	return false
}

func primaryService(d *detector.DetectionReport) *detector.ServiceState {
	if d == nil {
		return nil
	}
	if strings.TrimSpace(d.PrimaryService) != "" {
		for i := range d.Services {
			if strings.EqualFold(d.Services[i].Name, d.PrimaryService) && d.Services[i].Exists {
				return &d.Services[i]
			}
		}
	}
	for i := range d.Services {
		if d.Services[i].Exists {
			return &d.Services[i]
		}
	}
	return nil
}

func anyServiceExists(d *detector.DetectionReport) bool {
	for _, service := range d.Services {
		if service.Exists {
			return true
		}
	}
	return false
}

func anyFileExists(d *detector.DetectionReport) bool {
	for _, file := range d.Files {
		if file.Exists {
			return true
		}
	}
	return false
}

func anyRegistryExists(d *detector.DetectionReport) bool {
	for _, key := range d.Registry {
		if key.Exists {
			return true
		}
	}
	return false
}

func anyMSIRegistryExists(d *detector.DetectionReport) bool {
	for _, key := range d.Registry {
		if key.Exists && (strings.Contains(key.Path, `Installer\`) || strings.Contains(key.Path, `Uninstall\{EB1FBC37`)) {
			return true
		}
	}
	return false
}

func serviceRunning(d *detector.DetectionReport) bool {
	service := primaryService(d)
	return service != nil && strings.EqualFold(service.Status, "running")
}

func serviceStatus(d *detector.DetectionReport) string {
	service := primaryService(d)
	if service == nil {
		return ""
	}
	if service.Status != "" {
		return service.Status
	}
	return "exists"
}

func verificationCheckStatus(v any, name string, statuses ...string) bool {
	for _, check := range verificationChecks(v) {
		if !strings.EqualFold(check.name, name) {
			continue
		}
		for _, status := range statuses {
			if check.status == status {
				return true
			}
		}
	}
	return false
}

func verificationEvidence(v any) []string {
	evidence := []string{"Verification status: " + verificationOverallStatus(v)}
	for _, check := range verificationChecks(v) {
		if check.status == verificationPassed || check.status == verificationSkipped {
			continue
		}
		text := check.name + ": " + check.status
		if check.message != "" {
			text += " (" + check.message + ")"
		}
		evidence = append(evidence, text)
	}
	return evidence
}

type verificationCheck struct {
	name    string
	status  string
	message string
}

func verificationOverallStatus(v any) string {
	return verificationStringField(v, "OverallStatus")
}

func verificationChecks(v any) []verificationCheck {
	value := indirectValue(v)
	if !value.IsValid() || value.Kind() != reflect.Struct {
		return nil
	}
	field := value.FieldByName("Checks")
	if !field.IsValid() || field.Kind() != reflect.Slice {
		return nil
	}
	checks := make([]verificationCheck, 0, field.Len())
	for i := 0; i < field.Len(); i++ {
		checkValue := indirectValue(field.Index(i).Interface())
		if !checkValue.IsValid() || checkValue.Kind() != reflect.Struct {
			continue
		}
		checks = append(checks, verificationCheck{
			name:    stringField(checkValue, "Name"),
			status:  stringField(checkValue, "Status"),
			message: stringField(checkValue, "Message"),
		})
	}
	return checks
}

func verificationStringField(v any, name string) string {
	value := indirectValue(v)
	if !value.IsValid() || value.Kind() != reflect.Struct {
		return ""
	}
	return stringField(value, name)
}

func stringField(value reflect.Value, name string) string {
	field := value.FieldByName(name)
	if !field.IsValid() {
		return ""
	}
	switch field.Kind() {
	case reflect.String:
		return field.String()
	default:
		if field.CanInterface() {
			return fmt.Sprint(field.Interface())
		}
		return ""
	}
}

func indirectValue(v any) reflect.Value {
	if v == nil {
		return reflect.Value{}
	}
	value := reflect.ValueOf(v)
	for value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return reflect.Value{}
		}
		value = value.Elem()
	}
	return value
}

func existingFileEvidence(d *detector.DetectionReport) []string {
	var evidence []string
	for _, file := range d.Files {
		if file.Exists {
			evidence = append(evidence, file.Type+": "+file.Path)
		}
	}
	return evidence
}

func existingRegistryEvidence(d *detector.DetectionReport, msiOnly bool) []string {
	var evidence []string
	for _, key := range d.Registry {
		if !key.Exists {
			continue
		}
		if msiOnly && !(strings.Contains(key.Path, `Installer\`) || strings.Contains(key.Path, `Uninstall\{EB1FBC37`)) {
			continue
		}
		evidence = append(evidence, key.Root+`\`+key.Path)
	}
	return evidence
}

func wmiFilesExist(d *detector.DetectionReport) bool {
	for _, file := range d.Files {
		if file.Exists && pathLooksWMI(file.Path) {
			return true
		}
	}
	return false
}

func pathLooksWMI(path string) bool {
	normalized := strings.ToLower(detector.NormalizeWindowsPath(path))
	if normalized == "" || normalized == "." {
		return false
	}
	return strings.Contains(normalized, `\system32\wmi\`) || strings.HasSuffix(normalized, `\system32\wmi`) || strings.Contains(strings.ToLower(filepath.ToSlash(normalized)), "/system32/wmi/")
}

func prefixValues(prefix string, values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		result = append(result, prefix+value)
	}
	return result
}

func compact(values []string) []string {
	seen := map[string]bool{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		result = append(result, value)
	}
	return result
}

func valueOrDash(value string) string {
	if strings.TrimSpace(value) == "" {
		return "-"
	}
	return value
}
