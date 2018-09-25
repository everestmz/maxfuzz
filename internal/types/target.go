package types

type Target struct {
	Name     string `json:"name"`
	UniqueID string `json:"id"`
	Language string `json:"language"`
	Location string `json:"location"`
	Revision string `json:"revision"`
}
