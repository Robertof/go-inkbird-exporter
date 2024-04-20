package device

type Factory interface {
	FromSpec(spec DeviceSpec) (Device, error)
}

type FactoryDocs interface {
	Help() string
}
