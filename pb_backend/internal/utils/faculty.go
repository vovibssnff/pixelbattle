package utils

// ValidFaculties lists allowed faculty codes (aligned with benchmarks anonymous WS).
var ValidFaculties = map[string]struct{}{
	"KTU":  {},
	"TINT": {},
	"FTMF": {},
	"FTMI": {},
	"NOZH": {},
}

// IsValidFaculty reports whether faculty is an allowed value.
func IsValidFaculty(faculty string) bool {
	_, ok := ValidFaculties[faculty]
	return ok
}
