package external

//ProvisionerExternal simply wraps an existing kubernetes cluster (accessed via kubectl) and registry
type ProvisionerExternal struct{}

func (ProvisionerExternal) EnsureCluster() error {
	panic("implement me")
}

func (ProvisionerExternal) KubeConfigLocation() string {
	panic("implement me")
}

func (ProvisionerExternal) Registry() (string, error) {
	panic("implement me")
}

func (ProvisionerExternal) EdgeNodes() ([]string, error) {
	panic("implement me")
}

func (ProvisionerExternal) InClusterDir(hostDir string) string {
	panic("implement me")
}

//Create returns a new ProvisionerLocalDev from the given arguments
//noinspection GoUnusedParameter
func Create(map[string]string) *ProvisionerExternal {
	return &ProvisionerExternal{}
}