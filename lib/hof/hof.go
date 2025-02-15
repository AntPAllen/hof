/*
This package manages the #hof: _ configuration
and the various @<hof>() attributes
*/
package hof

// #hof: ...
type Hof struct {
	// label from containing value in CUE
	Label string

	// path to containing value in CUE
	Path  string

	// Semantic hof datamodel api-version for compat
	APIVersion string

	// #hof: metadata: ...
	Metadata  Metadata

	// #hof: <feature>: ...
	// @<feature>(<name>) can be shorthand with no-name implying label?
	Datamodel Datamodel
	Gen       Gen
	Flow      Flow
	Chat      Chat

	// any extra config, set by users
	Extra map[string]any
}

// Common Metadata
type Metadata struct {
	// A unique ID as @id(string) generated by hof
	ID       string `json:"id"`

	// Given name for the value
	Name     string

	// arbitrary key=string data
	Labels map[string]string

	// CUE import path / package
	// only needs to be set if importing elsewhere
	Package string
}

// hof/datamodel configuration
type Datamodel struct {
	// Root for datamodel features
	// This acts as the entrypoint
	// the other fields determine behavior
	Root bool

	// index, pivot, level, hierachy, node?
	Node bool

	// Enable snapshots and history in .hof/dm/...
	History bool

	// create an Ordered<Label> list in parent data
	// where elements are from CUE iteration of fields
	Ordered bool

	// treat as a cue value
	// nesting stops here and the whole
	// value is tracked as a singular value
	// support incomplete values & schemas
	Cue     bool

	// Semantic version for value
	Version string

}

// hof/gen configuration 
// #hof: gen: "name"
// @gen(name)
type Gen struct {
	Root    bool
	Name    string
	Creator bool
}

/*
  hof/flow configuration
  this has more shorthands
	@flow() | @flow(name)
	@task(op) | @task(op,name)
*/
type Flow struct {
	Root bool
	Name string
	Task string
}

/*
  hof/chat configuration
*/
type Chat struct {
	Root  bool
	Name  string
	Extra string
}
