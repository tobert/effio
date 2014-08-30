package effio

type RunCmd struct {
	SuiteCmd
	RerunFlag bool
}

func (cmd *Cmd) ToRunCmd() RunCmd {
	sc := cmd.ToSuiteCmd()
	rc := RunCmd{sc, false}
	rc.FlagSet.BoolVar(&rc.RerunFlag, "rerun", false, "rerun tests even if output.json already exists")
	return rc
}

// effio run -suite <dir>
func (rc *RunCmd) Run() {
	rc.ParseArgs()
	suite := rc.LoadSuite()
	suite.Run(rc.PathFlag, rc.RerunFlag)
}
