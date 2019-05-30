package provisioners

type ProvisionerLocalDev struct{}

func (provisioner *ProvisionerLocalDev) EnsureCluster() error {
	return nil //TODO
}
