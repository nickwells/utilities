package main

import (
	"sort"

	"github.com/nickwells/param.mod/v5/param"
	"github.com/nickwells/param.mod/v5/param/paction"
	"github.com/nickwells/param.mod/v5/param/psetter"
	"github.com/nickwells/snippet.mod/snippet"
)

const (
	paramNameSnippetList = "snippets-list"
)

// snippetListParams holds the values needed to configure the snippet list
type snippetListParams struct {
	listSnippets bool
	listDirs     bool

	constraints []string
	parts       []string
	tags        []string
	hideIntro   bool
}

// addSnippetListParams returns a func that will add parameters concerned
// with listing snippets to the passed param.PSet. Parameter values are set
// in the supplied snippetListParams
func addSnippetListParams(slp *snippetListParams) func(ps *param.PSet) error {
	return func(ps *param.PSet) error {
		const snippetListParamGroup = "cmd-snippet-list"
		ps.AddGroup(snippetListParamGroup,
			"parameters relating to listing snippets.")

		const (
			paramNameSnippetListShort      = "snippet-list-short"
			paramNameSnippetListConstraint = "snippet-list-constraint"
			paramNameSnippetListPart       = "snippet-list-part"
			paramNameSnippetListTag        = "snippet-list-tag"
			paramNameSnippetListDir        = "snippet-list-dir"
		)

		ps.Add(paramNameSnippetList,
			psetter.Bool{Value: &slp.listSnippets},
			"list all the available snippets and exit, no program is run."+
				" It will also show any per-snippet documentation and"+
				" report on any problems detected with the snippets.",
			param.GroupName(snippetListParamGroup),
			param.AltName("snippet-list"),
			param.AltName("s-l"),
			param.AltName("sl"),
			param.SeeAlso(
				paramNameSnippetListShort,
				paramNameSnippetListConstraint,
				paramNameSnippetListPart,
				paramNameSnippetListTag,
			),
			param.Attrs(param.CommandLineOnly),
		)

		ps.Add(paramNameSnippetListShort, psetter.Bool{Value: &slp.hideIntro},
			"this will prevent the printing of the part names before"+
				" the text of the snippet part. This can be useful if"+
				" you want to use the result in a script in which case"+
				" you will probably want to limit the parts shown as"+
				" well."+
				"\n\n"+
				"Setting this will also set the flag indicating that a"+
				" snippet list is wanted",
			param.GroupName(snippetListParamGroup),
			param.AltName("sl-short"),
			param.AltName("sl-s"),
			param.PostAction(paction.SetBool(&slp.listSnippets, true)),
			param.Attrs(param.CommandLineOnly|param.DontShowInStdUsage),
		)

		ps.Add(paramNameSnippetListConstraint,
			psetter.StrList{Value: &slp.constraints},
			"this restricts the snippets to show."+
				" The constraints can be a snippet name,"+
				" a snippet sub-directory"+
				" or a full pathname of either a file or directory."+
				"\n\n"+
				"Setting this will also set the flag indicating that a"+
				" snippet list is wanted",
			param.GroupName(snippetListParamGroup),
			param.AltName("sl-c"),
			param.PostAction(paction.SetBool(&slp.listSnippets, true)),
			param.Attrs(param.CommandLineOnly|param.DontShowInStdUsage),
		)

		validParts := psetter.AllowedVals(snippet.ValidParts())
		allParts, _ := validParts.Keys()
		sort.Strings(allParts)

		ps.Add(paramNameSnippetListPart,
			psetter.EnumList{
				Value:       &slp.parts,
				AllowedVals: validParts,
				Aliases: psetter.Aliases{
					"all": allParts,
				},
			},
			"this sets the parts of the snippet to show."+
				"\n\n"+
				"Setting this will also set the flag indicating that a"+
				" snippet list is wanted",
			param.GroupName(snippetListParamGroup),
			param.AltName("sl-p"),
			param.PostAction(paction.SetBool(&slp.listSnippets, true)),
			param.Attrs(param.CommandLineOnly|param.DontShowInStdUsage),
		)

		ps.Add(paramNameSnippetListTag,
			psetter.StrList{Value: &slp.tags},
			"this set the tags of the snippet to show."+
				"\n\n"+
				"Setting this will also set the flag indicating that a"+
				" snippet list is wanted",
			param.GroupName(snippetListParamGroup),
			param.AltName("sl-t"),
			param.PostAction(paction.SetBool(&slp.listSnippets, true)),
			param.Attrs(param.CommandLineOnly|param.DontShowInStdUsage),
		)

		ps.Add(paramNameSnippetListDir, psetter.Bool{Value: &slp.listDirs},
			"show the snippet directories",
			param.GroupName(snippetListParamGroup),
			param.AltName("sl-d"),
			param.Attrs(param.CommandLineOnly|param.DontShowInStdUsage),
		)

		return nil
	}
}
