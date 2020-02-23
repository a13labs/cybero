package core

// ServiceRequest represents a incoming request
type ServiceRequest struct {
	Module     string
	Action     string
	Parameters map[string]interface{}
}

// ServiceResponse represents a outgoing response
type ServiceResponse struct {
	Status   int
	Response map[string]interface{}
}

// ServiceModule represents a module plugin
type ServiceModule interface {
	Name() string
	Version() string
	Info() string
	Help(action string) string
	Handle(request *ServiceRequest) (int, map[string]interface{})
}

// ServiceAction internal callback signature
type ServiceAction map[string]func(*ServiceRequest) (int, map[string]interface{})
