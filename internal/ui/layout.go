package ui

const (
	minWindowWidth  = 1100
	minWindowHeight = 760
)

type rect struct {
	X int
	Y int
	W int
	H int
}

type windowLayout struct {
	Header        rect
	SystemGroup   rect
	System        rect
	QuickGroup    rect
	RepairGroup   rect
	DangerGroup   rect
	TimelineGroup rect
	Timeline      rect
	DetailsGroup  rect
	Details       rect
	Buttons       map[int]rect
	Labels        map[int]rect
}

func computeMainLayout(width int, height int) windowLayout {
	if width < minWindowWidth {
		width = minWindowWidth
	}
	if height < minWindowHeight {
		height = minWindowHeight
	}
	const margin = 16
	const gap = 12
	contentW := width - margin*2
	layout := windowLayout{
		Buttons: map[int]rect{},
		Labels:  map[int]rect{},
	}

	layout.Header = rect{margin, 12, contentW, 82}
	layout.SystemGroup = rect{margin, 104, contentW, 146}
	layout.System = rect{margin + 16, 130, contentW - 32, 104}
	layout.QuickGroup = rect{margin, 260, contentW, 70}

	x := margin + 16
	y := 286
	for _, spec := range []struct {
		id int
		w  int
	}{
		{idCheck, 100},
		{idVerify, 100},
		{idCollect, 130},
		{idOpenReports, 170},
		{idRestartAdmin, 190},
		{idCancel, 100},
	} {
		layout.Buttons[spec.id] = rect{x, y, spec.w, 30}
		x += spec.w + 8
	}

	panelY := 342
	panelH := 156
	dangerW := 392
	repairW := contentW - gap - dangerW
	if repairW < 620 {
		repairW = contentW
		dangerW = contentW
	}
	layout.RepairGroup = rect{margin, panelY, repairW, panelH}
	if repairW == contentW {
		layout.DangerGroup = rect{margin, panelY + panelH + gap, dangerW, panelH}
	} else {
		layout.DangerGroup = rect{margin + repairW + gap, panelY, dangerW, panelH}
	}

	rx := layout.RepairGroup.X
	rw := layout.RepairGroup.W
	layout.Labels[idStaticInstaller] = rect{rx + 16, panelY + 28, 90, 22}
	layout.Buttons[idBrowseInstaller] = rect{rx + rw - 124, panelY + 24, 96, 28}
	layout.Buttons[idInstaller] = rect{rx + 110, panelY + 26, rw - 250, 24}
	layout.Labels[idStaticInvite] = rect{rx + 16, panelY + 62, 90, 22}
	layout.Buttons[idInvite] = rect{rx + 110, panelY + 60, 250, 24}
	layout.Labels[idStaticProfile] = rect{rx + 382, panelY + 62, 55, 22}
	layout.Buttons[idProfile] = rect{rx + 440, panelY + 60, maxInt(120, rw-468), 24}
	layout.Labels[idStaticProfiles] = rect{rx + 110, panelY + 92, rw - 140, 22}
	layout.Buttons[idPreflight] = rect{rx + 16, panelY + 118, 112, 28}
	layout.Buttons[idRepairDryRun] = rect{rx + 138, panelY + 118, 140, 28}

	dx := layout.DangerGroup.X
	dw := layout.DangerGroup.W
	layout.Labels[idStaticDanger1] = rect{dx + 18, layout.DangerGroup.Y + 28, dw - 36, 38}
	layout.Labels[idStaticDanger2] = rect{dx + 18, layout.DangerGroup.Y + 70, dw - 36, 24}
	layout.Buttons[idRepair] = rect{dx + 18, layout.DangerGroup.Y + 106, 190, 38}

	timelineY := maxInt(layout.RepairGroup.Y+layout.RepairGroup.H, layout.DangerGroup.Y+layout.DangerGroup.H) + gap
	remainingH := height - timelineY - margin
	detailsW := maxInt(392, contentW*38/100)
	timelineW := contentW - gap - detailsW
	if timelineW < 520 {
		timelineW = contentW
		detailsW = contentW
	}
	layout.TimelineGroup = rect{margin, timelineY, timelineW, remainingH}
	if timelineW == contentW {
		layout.DetailsGroup = rect{margin, timelineY + remainingH + gap, detailsW, 260}
	} else {
		layout.DetailsGroup = rect{margin + timelineW + gap, timelineY, detailsW, remainingH}
	}
	layout.Timeline = rect{layout.TimelineGroup.X + 16, timelineY + 26, layout.TimelineGroup.W - 32, remainingH - 42}

	buttonY := layout.DetailsGroup.Y + layout.DetailsGroup.H - 80
	detailsH := maxInt(80, buttonY-(layout.DetailsGroup.Y+26)-10)
	layout.Details = rect{layout.DetailsGroup.X + 16, layout.DetailsGroup.Y + 26, layout.DetailsGroup.W - 32, detailsH}
	bx := layout.DetailsGroup.X + 16
	layout.Buttons[idOpenSelectedReport] = rect{bx, buttonY, 124, 28}
	layout.Buttons[idOpenOperations] = rect{bx + 132, buttonY, 128, 28}
	layout.Buttons[idCopyReportPath] = rect{bx + 268, buttonY, 112, 28}
	layout.Buttons[idReportsList] = rect{bx, buttonY + 36, 118, 28}
	layout.Buttons[idReportsCleanupDry] = rect{bx + 126, buttonY + 36, 144, 28}
	layout.Buttons[idCopyBundlePath] = rect{bx + 278, buttonY + 36, 102, 28}
	return layout
}

func maxInt(a int, b int) int {
	if a > b {
		return a
	}
	return b
}
