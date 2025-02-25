package utils

/// ****** R ****** ///
type R map[int]map[int]map[int]bool

func InitR() R {
	return make(R)
}

// setR sets the value of r[l][j][k] in R.
func (r R) setR(l, j, k int, value bool) {
	if _, exists := r[l]; !exists {
		r[l] = make(map[int]map[int]bool)
	}
	if _, exists := r[l][j]; !exists {
		r[l][j] = make(map[int]bool)
	}
	r[l][j][k] = value
}

// getR gets the value of r[l][j][k], defaulting to false if not set.
func (r R) getR(l, j, k int) bool {
	if _, exists := r[l]; exists {
		if _, exists := r[l][j]; exists {
			if val, exists := r[l][j][k]; exists {
				return val
			}
		}
	}
	return false
}

/// ****** X ****** ///
type X map[int]map[int]int

func InitX() X {
	return make(X)
}

// setR sets the value of r[l][j][k] in R.
func (r R) setR(l, j, k int, value bool) {
	if _, exists := r[l]; !exists {
		r[l] = make(map[int]map[int]bool)
	}
	if _, exists := r[l][j]; !exists {
		r[l][j] = make(map[int]bool)
	}
	r[l][j][k] = value
}

// getR gets the value of r[l][j][k], defaulting to false if not set.
func (r R) getR(l, j, k int) bool {
	if _, exists := r[l]; exists {
		if _, exists := r[l][j]; exists {
			if val, exists := r[l][j][k]; exists {
				return val
			}
		}
	}
	return false
}

/// ****** F ****** ///
// Function represents a function provided by a microservice
type Function struct {
	FunctionID  int     // f_{i,j}
	InputSize   int     // datain_{i,j}
	OutputSize  int     // dataout_{i,j}
	EdgeTime    float64 // tp_{i,j} (processing time on edge)
	CloudTime   float64 // tp,C_{i,j} (processing time on cloud)
	ResourceReq int     // Required resources
}

// F represents the function set for microservices
type F map[int][]Function // Key: Microservice ID (s_i), Value: List of functions

// InitF initializes an empty function set
func InitF() F {
	return make(F)
}

// AddFunction adds a function to a microservice
func (f F) AddFunction(serviceID, functionID, inputSize, outputSize, resourceReq int, edgeTime, cloudTime float64) {
	f[serviceID] = append(f[serviceID], Function{
		FunctionID:  functionID,
		InputSize:   inputSize,
		OutputSize:  outputSize,
		EdgeTime:    edgeTime,
		CloudTime:   cloudTime,
		ResourceReq: resourceReq,
	})
}

// GetFunctions retrieves all functions for a microservice
func (f F) GetFunctions(serviceID int) []Function {
	return f[serviceID]
}

// GetFunction retrieves a specific function within a microservice
func (f F) GetFunction(serviceID, functionID int) *Function {
	for _, function := range f[serviceID] {
		if function.FunctionID == functionID {
			return &function
		}
	}
	return nil // Return nil if function not found
}

/// ****** S ****** ///
// Microservice represents a microservice with processing capacity and resource requirements
type Microservice struct {
	ID            int        // Unique Microservice ID (s_i)
	Name          string     // Name of the Microservice
	ProcessingCap int        // μ: Max number of requests per unit time
	ResourceReq   int        // r: CPU/RAM required for an instance
	Functions     []Function // Functions belonging to this microservice
}

// S represents the set of microservices
type S map[int]Microservice

// InitS initializes an empty microservice set
func InitS() S {
	return make(S)
}

// AddMicroservice adds a microservice to the set
func (s S) AddMicroservice(id int, name string, processingCap, resourceReq int) {
	s[id] = Microservice{
		ID:            id,
		Name:          name,
		ProcessingCap: processingCap,
		ResourceReq:   resourceReq,
		Functions:     []Function{},
	}
}

// AddFunctionToMicroservice adds a function to an existing microservice
func (s S) AddFunctionToMicroservice(microID int, function Function) {
	if micro, exists := s[microID]; exists {
		micro.Functions = append(micro.Functions, function)
		s[microID] = micro
	}
}

// GetMicroservice retrieves a microservice by ID
func (s S) GetMicroservice(id int) (Microservice, bool) {
	micro, exists := s[id]
	return micro, exists
}

/// ****** A ****** ///
type A map[int][]Function

func InitA() A {
	return make(A)
}

// addApp adds a new application with an ordered list of functions
func (a A) addApp(appID int, functions []Function) {
	a[appID] = functions
}

// addFunction adds a function to an existing application sequence
func (a A) addFunction(appID, serviceID, functionID int) {
	a[appID] = append(a[appID], Function{ServiceID: serviceID, FunctionID: functionID})
}

// getApp retrieves the function sequence for an application
func (a A) getApp(appID int) []Function {
	return a[appID]
}

// removeApp deletes an entire application from A
func (a A) removeApp(appID int) {
	delete(a, appID)
}

// removeFunction removes a function at a given position in an application sequence
func (a A) removeFunction(appID, index int) {
	if _, exists := a[appID]; exists && index < len(a[appID]) {
		a[appID] = append(a[appID][:index], a[appID][index+1:]...)
	}
}
