package ssa

import "golang.org/x/tools/go/pointer"

// PtrAnlysCfg returns a default pointer analysis config from Info.
func (info *Info) PtrAnlysCfg(tests bool) (*pointer.Config, error) {
	mains, err := MainPkgs(info.Prog, tests)
	if err != nil {
		return nil, err
	}
	return &pointer.Config{
		Mains:      mains,
		Log:        info.PtaLog,
		Reflection: false,
	}, nil
}

// RunPtrAnlys runs pointer analysis and returns the analysis result.
func (info *Info) RunPtrAnlys(config *pointer.Config) (*pointer.Result, error) {
	return pointer.Analyze(config)
}
