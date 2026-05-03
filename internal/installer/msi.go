package installer

func MSIInstallPlan(installerPath string) string {
	if installerPath == "" {
		return "MSI install placeholder; no installer path provided"
	}
	return "MSI install placeholder for " + installerPath
}
