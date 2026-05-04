package app

import (
	"errors"
	"reflect"
	"testing"
)

type fakeAdminChecker struct {
	admin bool
}

func (c fakeAdminChecker) IsAdmin() bool {
	return c.admin
}

type fakeElevationLauncher struct {
	called bool
	args   []string
	err    error
}

func (l *fakeElevationLauncher) RelaunchElevated(args []string) error {
	l.called = true
	l.args = append([]string(nil), args...)
	return l.err
}

func TestEnsureAdminOrRelaunch(t *testing.T) {
	tests := []struct {
		name         string
		opts         AdminGuardOptions
		wantRelaunch bool
		wantCode     int
		wantLauncher bool
		wantErr      bool
	}{
		{
			name: "requires admin false does nothing",
			opts: AdminGuardOptions{RequiresAdmin: false, AdminChecker: fakeAdminChecker{}},
		},
		{
			name: "already admin does nothing",
			opts: AdminGuardOptions{RequiresAdmin: true, AdminChecker: fakeAdminChecker{admin: true}},
		},
		{
			name:         "non admin no elevate returns code 2",
			opts:         AdminGuardOptions{CommandName: "repair", RequiresAdmin: true, NoElevate: true, AdminChecker: fakeAdminChecker{}},
			wantCode:     ExitAdminRequired,
			wantErr:      true,
			wantLauncher: false,
		},
		{
			name:     "non admin quiet returns code 2",
			opts:     AdminGuardOptions{RequiresAdmin: true, Quiet: true, AdminChecker: fakeAdminChecker{}},
			wantCode: ExitAdminRequired,
			wantErr:  true,
		},
		{
			name:     "non admin non interactive returns code 2",
			opts:     AdminGuardOptions{RequiresAdmin: true, NonInteractive: true, AdminChecker: fakeAdminChecker{}},
			wantCode: ExitAdminRequired,
			wantErr:  true,
		},
		{
			name:     "elevated child still non admin returns code 2",
			opts:     AdminGuardOptions{RequiresAdmin: true, ElevatedChild: true, AdminChecker: fakeAdminChecker{}},
			wantCode: ExitAdminRequired,
			wantErr:  true,
		},
		{
			name:         "non admin interactive attempts relaunch",
			opts:         AdminGuardOptions{RequiresAdmin: true, Args: []string{"repair"}, AdminChecker: fakeAdminChecker{}},
			wantRelaunch: true,
			wantLauncher: true,
		},
		{
			name:         "failed relaunch returns code 10",
			opts:         AdminGuardOptions{RequiresAdmin: true, Args: []string{"repair"}, AdminChecker: fakeAdminChecker{}},
			wantCode:     ExitUnexpectedError,
			wantLauncher: true,
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			launcher := &fakeElevationLauncher{}
			if tt.name == "failed relaunch returns code 10" {
				launcher.err = errors.New("cancelled")
			}
			tt.opts.Launcher = launcher
			got := EnsureAdminOrRelaunch(tt.opts)
			if got.Relaunched != tt.wantRelaunch {
				t.Fatalf("Relaunched = %t, want %t", got.Relaunched, tt.wantRelaunch)
			}
			if got.ExitCode != tt.wantCode {
				t.Fatalf("ExitCode = %d, want %d", got.ExitCode, tt.wantCode)
			}
			if (got.Err != nil) != tt.wantErr {
				t.Fatalf("Err presence = %t, want %t", got.Err != nil, tt.wantErr)
			}
			if launcher.called != tt.wantLauncher {
				t.Fatalf("launcher called = %t, want %t", launcher.called, tt.wantLauncher)
			}
		})
	}
}

func TestAppendElevatedChildArg(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want []string
	}{
		{name: "appends missing flag", args: []string{"repair"}, want: []string{"repair", ElevatedChildFlag}},
		{name: "keeps existing flag", args: []string{"repair", ElevatedChildFlag}, want: []string{"repair", ElevatedChildFlag}},
		{name: "keeps existing assigned flag", args: []string{"repair", ElevatedChildFlag + "=true"}, want: []string{"repair", ElevatedChildFlag + "=true"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := AppendElevatedChildArg(tt.args)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("AppendElevatedChildArg() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestEnsureAdminOrRelaunchPassesElevatedChildOnce(t *testing.T) {
	launcher := &fakeElevationLauncher{}
	result := EnsureAdminOrRelaunch(AdminGuardOptions{
		RequiresAdmin: true,
		Args:          []string{"repair", ElevatedChildFlag},
		AdminChecker:  fakeAdminChecker{},
		Launcher:      launcher,
	})
	if !result.Relaunched {
		t.Fatalf("Relaunched = false, want true")
	}
	want := []string{"repair", ElevatedChildFlag}
	if !reflect.DeepEqual(launcher.args, want) {
		t.Fatalf("launcher args = %#v, want %#v", launcher.args, want)
	}
}

func TestMaskSensitiveArgs(t *testing.T) {
	args := []string{
		"repair",
		"--invite",
		"SECRET",
		"--installer",
		`C:\Installers\Kickidler Grabber\grabber.msi`,
		"--invite=SECRET2",
	}
	got := MaskSensitiveArgs(args)
	want := []string{
		"repair",
		"--invite",
		"*****",
		"--installer",
		`C:\Installers\Kickidler Grabber\grabber.msi`,
		"--invite=*****",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("MaskSensitiveArgs() = %#v, want %#v", got, want)
	}
	if reflect.DeepEqual(args, got) {
		t.Fatalf("MaskSensitiveArgs should not expose raw invite values")
	}
}
