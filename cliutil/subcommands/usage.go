package subcommands

// Description contains a short description of the command it is
// embedded in.
type Description string

var _ DescriptionGetter = Description("")

// GetDescription returns the description. See DescriptionGetter.
func (d Description) GetDescription() string {
	return string(d)
}

// Synopses contains a list of synopses snippets, short summaries of
// the arguments that can be passed in.
//
// If you have just one synopsis, use Synopsis instead.
type Synopses []string

var _ SynopsesGetter = Synopses{}

// GetSynopses returns the list of synopses. See SynopsesGetter.
func (s Synopses) GetSynopses() []string {
	return s
}

// Synopsis contains a synopsis snippet, a short summary of the
// arguments that can be passed in.
//
// To show multiple alternative calling conventions, use Synopses.
type Synopsis string

var _ SynopsesGetter = Synopsis("")

// GetSynopses returns the list of synopses. See SynopsesGetter.
func (s Synopsis) GetSynopses() []string {
	return []string{string(s)}
}

// Overview contains one or more paragraphs of text giving an overview
// of the command.
type Overview string

var _ Overviewer = Overview("")

// GetOverview returns the overview text. See Overviewer.
func (s Overview) GetOverview() string {
	return string(s)
}
