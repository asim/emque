package server

type Server interface {
	Run() error
	Stop() error
}
