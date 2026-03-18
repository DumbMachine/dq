package types

type AnnotationFile struct {
	Tables map[string]AnnotationTable `yaml:"tables" json:"tables"`
}

type AnnotationTable struct {
	Note    string                      `yaml:"note,omitempty" json:"note,omitempty"`
	Columns map[string]AnnotationColumn `yaml:"columns,omitempty" json:"columns,omitempty"`
}

type AnnotationColumn struct {
	Note string `yaml:"note" json:"note"`
}
