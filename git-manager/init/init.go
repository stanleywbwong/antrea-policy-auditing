package init

import (
    "os"
	"io/ioutil"

	. "antrea-audit/git-manager/gitops"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"

	billy "github.com/go-git/go-billy/v5"
	"github.com/go-git/go-git/v5/storage/memory"
	"github.com/ghodss/yaml"
    "github.com/go-git/go-git/v5"
)

func SetupRepo(k *Kubernetes, dir *string) error {
	r, err := createRepo(k, dir)
	if err != nil {
		klog.ErrorS(err, "unable to create network policy repository")
		return err
	}
	if err := addResources(k, *dir); err != nil {
		klog.ErrorS(err, "unable to add resource yamls to repository")
		return err
	}
	if err := AddAndCommit(r, "audit-init", "system@audit.antrea.io", "Initial commit of existing policies"); err != nil {
		klog.ErrorS(err, "unable to add and commit existing resources to repository")
		return err
	}
	klog.V(2).Infof("Repository successfully initialized at %s", *dir)
	return nil
}

func createRepo(k *Kubernetes, dir *string) (*git.Repository, error) {
    if *dir == "" {
        path, err := os.Getwd()
        if err != nil {
			klog.ErrorS(err, "unable to retrieve the current working directory")
			return nil, err
        }
        *dir = path
    }
	*dir += "/network-policy-repository"
    r, err := git.PlainInit(*dir, false)
    if err == git.ErrRepositoryAlreadyExists {
		klog.ErrorS(err, "network policy respository already exists - skipping initialization")
		return nil, err
	} else if err != nil {
		klog.ErrorS(err, "unable to initialize git repo")
		return nil, err
	}
	return r, nil
}

func SetupRepoInMem(k *Kubernetes, storer *memory.Storage, fs billy.Filesystem) error {
	r, err := git.Init(storer, fs)
    if err == git.ErrRepositoryAlreadyExists {
		klog.ErrorS(err, "network policy respository already exists - skipping initialization")
		return err
	} else if err != nil {
		klog.ErrorS(err, "unable to initialize git repo")
		return err
	}
	if err := addResourcesInMem(k, fs); err != nil {
		klog.ErrorS(err, "unable to write network policies to repository")
		return err
	}
	if err := AddAndCommit(r, "audit-init", "system@audit.antrea.io", "initial commit of existing policies"); err != nil {
		klog.ErrorS(err, "unable to add and commit existing policies to repository")
		return err		
	}
	klog.V(2).Infof("Repository successfully initialized")
	return nil
}

func addResources(k *Kubernetes, dir string) error {
    os.Mkdir(dir + "/k8s-policies", 0700)
    os.Mkdir(dir + "/antrea-policies", 0700)
    os.Mkdir(dir + "/antrea-cluster-policies", 0700)
	os.Mkdir(dir + "/antrea-tiers", 0700)
	if err := addK8sPolicies(k, dir); err != nil {
		klog.ErrorS(err, "unable to add K8s network policies to repository")
		return err
	}
	if err := addAntreaPolicies(k, dir); err != nil {
		klog.ErrorS(err, "unable to add Antrea network policies to repository")
		return err
	}
	if err := addAntreaClusterPolicies(k, dir); err != nil {
		klog.ErrorS(err, "unable to add Antrea cluster network policies to repository")
		return err
	}
	if err := addAntreaTiers(k, dir); err != nil {
		klog.ErrorS(err, "unable to add Antrea tiers to repository")
		return err
	}
	return nil
}

func addResourcesInMem(k *Kubernetes, fs billy.Filesystem) error {
	fs.MkdirAll("k8s-policies", 0700)
	fs.MkdirAll("antrea-policies", 0700)
	fs.MkdirAll("antrea-cluster-policies", 0700)
	fs.MkdirAll("antrea-tiers", 0700)
	if err := addK8sPoliciesInMem(k, fs); err != nil {
		klog.ErrorS(err, "unable to add K8s network policies to repository")
		return err
	}
	if err := addAntreaPoliciesInMem(k, fs); err != nil {
		klog.ErrorS(err, "unable to add Antrea network policies to repository")
		return err
	}
	if err := addAntreaClusterPoliciesInMem(k, fs); err != nil {
		klog.ErrorS(err, "unable to add Antrea cluster network policies to repository")
		return err
	}
	if err := addAntreaTiersInMem(k, fs); err != nil {
		klog.ErrorS(err, "unable to add Antrea tiers to repository")
		return err
	}
	return nil
}

func addK8sPolicies(k *Kubernetes, dir string) error {
	policies, err := k.GetK8sPolicies()
	if err != nil {
		return err
	}
	var namespaces []string
	for _, np := range policies.Items {
		np.TypeMeta = metav1.TypeMeta{
			Kind: "NetworkPolicy",
			APIVersion: "networking.k8s.io/v1",
		}
		if !stringInSlice(np.Namespace, namespaces) {
			namespaces = append(namespaces, np.Namespace)
			os.Mkdir(dir + "/k8s-policies/" + np.Namespace, 0700)
		}
		path := dir + "/k8s-policies/" + np.Namespace + "/" + np.Name + ".yaml"
		klog.V(2).Infof("Added K8s policy at network-policy-repository/k8s-policies/" + np.Namespace + "/" + np.Name + ".yaml")
		y, err := yaml.Marshal(&np)
		if err != nil {
			klog.ErrorS(err, "unable to marshal policy config")
			return err
		}
		err = ioutil.WriteFile(path, y, 0644)
		if err != nil {
			klog.ErrorS(err, "unable to write policy config to file")
			return err
		}
	}
	return nil
}

func addK8sPoliciesInMem(k *Kubernetes, fs billy.Filesystem) error {
	policies, err := k.GetK8sPolicies()
	if err != nil {
		return err
	}
	var namespaces []string
	for _, np := range policies.Items {
		np.TypeMeta = metav1.TypeMeta{
			Kind: "NetworkPolicy",
			APIVersion: "networking.k8s.io/v1",
		}
		if !stringInSlice(np.Namespace, namespaces) {
			namespaces = append(namespaces, np.Namespace)
			fs.MkdirAll("k8s-policies/" + np.Namespace, 0700)
		}
		path := "k8s-policies/" + np.Namespace + "/" + np.Name + ".yaml"
		klog.V(2).Infof("Added K8s policy at k8s-policies/" + np.Namespace + "/" + np.Name + ".yaml")
		y, err := yaml.Marshal(&np)
		if err != nil {
			klog.ErrorS(err, "unable to marshal policy config")
			return err
		}
		newFile, err := fs.Create(path)
		if err != nil {
			klog.ErrorS(err, "unable to write policy config to file")
			return err
		}
		newFile.Write(y)
		newFile.Close()
	}
	return nil
}

func addAntreaPolicies(k *Kubernetes, dir string) error {
	policies, err := k.GetAntreaPolicies()
	if err != nil {
		return err
	}
	var namespaces []string
	for _, np := range policies.Items {
		np.TypeMeta = metav1.TypeMeta{
			Kind: "NetworkPolicy",
			APIVersion: "crd.antrea.io/v1alpha1",
		}
		if !stringInSlice(np.Namespace, namespaces) {
			namespaces = append(namespaces, np.Namespace)
			os.Mkdir(dir + "/antrea-policies/" + np.Namespace, 0700)
		}
		path := dir + "/antrea-policies/" + np.Namespace + "/" + np.Name + ".yaml"
		klog.V(2).Infof("Added Antrea policy at network-policy-repository/antrea-policies/" + np.Namespace + "/" + np.Name + ".yaml")
		y, err := yaml.Marshal(&np)
		if err != nil {
			klog.ErrorS(err, "unable to marshal policy config")
			return err
		}
		err = ioutil.WriteFile(path, y, 0644)
		if err != nil {
			klog.ErrorS(err, "unable to write policy config to file")
			return err
		}
	}
	return nil
}

func addAntreaPoliciesInMem(k *Kubernetes, fs billy.Filesystem) error {
	policies, err := k.GetAntreaPolicies()
	if err != nil {
		return err
	}
	var namespaces []string
	for _, np := range policies.Items {
		np.TypeMeta = metav1.TypeMeta{
			Kind: "NetworkPolicy",
			APIVersion: "crd.antrea.io/v1alpha1",
		}
		if !stringInSlice(np.Namespace, namespaces) {
			namespaces = append(namespaces, np.Namespace)
			fs.MkdirAll("antrea-policies/" + np.Namespace, 0700)
		}
		path := "antrea-policies/" + np.Namespace + "/" + np.Name + ".yaml"
		klog.V(2).Infof("Added Antrea policy at antrea-policies/" + np.Namespace + "/" + np.Name + ".yaml")
		y, err := yaml.Marshal(&np)
		if err != nil {
			klog.ErrorS(err, "unable to marshal policy config")
			return err
		}
		newFile, err := fs.Create(path)
		if err != nil {
			klog.ErrorS(err, "unable to write policy config to file")
			return err
		}
		newFile.Write(y)
		newFile.Close()
	}
	return nil
}

func addAntreaClusterPolicies(k *Kubernetes, dir string) error {
	policies, err := k.GetAntreaClusterPolicies()
	if err != nil {
		return err
	}
	for _, np := range policies.Items {
		np.TypeMeta = metav1.TypeMeta{
			Kind: "ClusterNetworkPolicy",
			APIVersion: "crd.antrea.io/v1alpha1",
		}
		path := dir + "/antrea-cluster-policies/" + np.Name + ".yaml"
		klog.V(2).Infof("Added Antrea cluster policy at network-policy-repository/antrea-cluster-policies/" + np.Name + ".yaml")
		y, err := yaml.Marshal(&np)
		if err != nil {
			klog.ErrorS(err, "unable to marshal policy config")
			return err
		}
		err = ioutil.WriteFile(path, y, 0644)
		if err != nil {
			klog.ErrorS(err, "unable to write policy config to file")
			return err
		}
	}
	return nil
}

func addAntreaClusterPoliciesInMem(k *Kubernetes, fs billy.Filesystem) error {
	policies, err := k.GetAntreaClusterPolicies()
	if err != nil {
		return err
	}
	for _, np := range policies.Items {
		np.TypeMeta = metav1.TypeMeta{
			Kind: "ClusterNetworkPolicy",
			APIVersion: "crd.antrea.io/v1alpha1",
		}
		path := "antrea-cluster-policies/" + np.Name + ".yaml"
		klog.V(2).Infof("Added Antrea cluster policy at network-policy-repository/antrea-cluster-policies/" + np.Name + ".yaml")
		y, err := yaml.Marshal(&np)
		if err != nil {
			klog.ErrorS(err, "unable to marshal policy config")
			return err
		}
		newFile, err := fs.Create(path)
		if err != nil {
			klog.ErrorS(err, "unable to write policy config to file")
			return err
		}
		newFile.Write(y)
		newFile.Close()
	}
	return nil
}

func addAntreaTiers(k *Kubernetes, dir string) error {
	tiers, err := k.GetAntreaTiers()
	if err != nil {
		return err
	}
	for _, tier := range tiers.Items {
		tier.TypeMeta = metav1.TypeMeta{
			Kind: "Tier",
			APIVersion: "crd.antrea.io/v1alpha1",
		}
		path := dir + "/antrea-tiers/" + tier.Name + ".yaml"
		klog.V(2).Infof("Added Antrea tier at network-policy-repository/antrea-tiers/" + tier.Name + ".yaml")
		y, err := yaml.Marshal(&tier)
		if err != nil {
			klog.ErrorS(err, "unable to marshal tier config")
			return err
		}
		err = ioutil.WriteFile(path, y, 0644)
		if err != nil {
			klog.ErrorS(err, "unable to write tier config to file")
			return err
		}
	}
	return nil
}

func addAntreaTiersInMem(k *Kubernetes, fs billy.Filesystem) error {
	tiers, err := k.GetAntreaTiers()
	if err != nil {
		return err
	}
	for _, tier := range tiers.Items {
		tier.TypeMeta = metav1.TypeMeta{
			Kind: "Tier",
			APIVersion: "crd.antrea.io/v1alpha1",
		}
		path := "antrea-tiers/" + tier.Name + ".yaml"
		klog.V(2).Infof("Added Antrea tier at network-policy-repository/antrea-tiers/" + tier.Name + ".yaml")
		y, err := yaml.Marshal(&tier)
		if err != nil {
			klog.ErrorS(err, "unable to marshal tier config")
			return err
		}
		newFile, err := fs.Create(path)
		if err != nil {
			klog.ErrorS(err, "unable to write tier config to file")
			return err
		}
		newFile.Write(y)
		newFile.Close()
	}
	return nil
}

func stringInSlice(a string, list []string) bool {
    for _, b := range list {
        if b == a {
            return true
        }
    }
    return false
}
