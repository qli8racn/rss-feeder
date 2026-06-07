package file

type FeedsReader interface {
	Load() ([]string, error)
}
