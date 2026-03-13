package engine

type StateSaver interface {
	Save(state *WorldState) error
	Load() (*WorldState, error)
}
