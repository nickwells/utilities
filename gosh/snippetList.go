package main

import (
	"maps"
	"slices"

	"github.com/nickwells/param.mod/v6/paction"
	"github.com/nickwells/param.mod/v6/param"
	"github.com/nickwells/param.mod/v6/psetter"
	"github.com/nickwells/snippet.mod/snippet"
)

const (
	paramNameSnippetList           = "snippet-list"
	paramNameSnippetListShort      = "snippet-list-short"
	paramNameSnippetListConstraint = "snippet-list-constraint"
	paramNameSnippetListPart       = "snippet-list-part"
	paramNameSnippetListTag        = "snippet-list-tag"
	paramNameSnippetListDir        = "snippet-list-dir"
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
	const snippetListNote = "\n\n" +
		"Setting this will also set the flag indicating that a" +
		" snippet list is wanted"

	return func(ps *param.PSet) error {
		const snippetListParamGroup = "cmd-snippet-list"

		ps.AddGroup(snippetListParamGroup,
			"parameters relating to listing snippets.")

		ps.Add(paramNameSnippetList,
			psetter.Bool{Value: &slp.listSnippets},
			"list all the available snippets and exit, no program is run."+
				" It will also show any per-snippet documentation and"+
				" report on any problems detected with the snippets.",
			param.GroupName(snippetListParamGroup),
			param.AltNames("snippets-list", "s-l", "sl",
				"snippets-show", "snippet-show"),
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
				" well."+snippetListNote,
			param.GroupName(snippetListParamGroup),
			param.AltNames("sl-short", "sl-s"),
			param.PostAction(paction.SetVal(&slp.listSnippets, true)),
			param.Attrs(param.CommandLineOnly|param.DontShowInStdUsage),
		)

		ps.Add(paramNameSnippetListConstraint,
			psetter.StrList[string]{Value: &slp.constraints},
			"this restricts the snippets to show. The constraints can be"+
				" a snippet name,"+
				" a snippet sub-directory"+
				" or a full pathname of either"+
				" a file or directory."+snippetListNote,
			param.GroupName(snippetListParamGroup),
			param.AltNames("sl-c", "snippet-list-only",
				"snippet-show-constraint", "snippet-show-only"),
			param.PostAction(paction.SetVal(&slp.listSnippets, true)),
			param.Attrs(param.CommandLineOnly|param.DontShowInStdUsage),
		)

		validParts := psetter.AllowedVals[string](snippet.ValidParts())
		allParts := slices.Sorted(maps.Keys(validParts))

		ps.Add(paramNameSnippetListPart,
			psetter.EnumList[string]{
				Value:       &slp.parts,
				AllowedVals: validParts,
				Aliases: psetter.Aliases[string]{
					"all": allParts,
				},
			},
			"this sets the parts of the snippet to show."+snippetListNote,
			param.GroupName(snippetListParamGroup),
			param.AltNames("sl-p", "snippet-list-parts",
				"snippet-show-part", "snippet-show-parts"),
			param.PostAction(paction.SetVal(&slp.listSnippets, true)),
			param.Attrs(param.CommandLineOnly|param.DontShowInStdUsage),
		)

		ps.Add(paramNameSnippetListTag,
			psetter.StrList[string]{Value: &slp.tags},
			"this sets the tags of the snippet to show."+
				" The value of a tag given here will be"+
				" shown when listing the snippets."+
				" The available tag names for a given"+
				" snippet may be found by showing the"+
				" Tags when selecting the parts of the"+
				" snippet to show."+snippetListNote,
			param.GroupName(snippetListParamGroup),
			param.AltNames("sl-t"),
			param.PostAction(paction.SetVal(&slp.listSnippets, true)),
			param.SeeAlso(paramNameSnippetListPart),
			param.Attrs(param.CommandLineOnly|param.DontShowInStdUsage),
		)

		ps.Add(paramNameSnippetListDir, psetter.Bool{Value: &slp.listDirs},
			"show the snippet directories",
			param.GroupName(snippetListParamGroup),
			param.AltNames("sl-d", "snippet-list-dirs",
				"snippet-show-dir", "snippet-show-dirs"),
			param.Attrs(param.CommandLineOnly|param.DontShowInStdUsage),
		)

		return nil
	}
}
