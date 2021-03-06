package util

import (
	"encoding/binary"
	"fmt"
	"hash/fnv"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"

	v1alpha1 "github.com/jetstack/navigator/pkg/apis/navigator/v1alpha1"
	hashutil "github.com/jetstack/navigator/pkg/util/hash"
)

const (
	NodePoolNameLabelKey      = "navigator.jetstack.io/elasticsearch-node-pool-name"
	NodePoolHashAnnotationKey = "navigator.jetstack.io/elasticsearch-node-pool-hash"
)

// ComputeHash returns a hash value calculated from pod template and a collisionCount to avoid hash collision
func ComputeNodePoolHash(c *v1alpha1.ElasticsearchCluster, np *v1alpha1.ElasticsearchClusterNodePool, collisionCount *int32) string {
	hashVar := struct {
		MinimumMasters int64
		ESImage        *v1alpha1.ImageSpec
		PilotImage     v1alpha1.ImageSpec
		Sysctl         []string
		Plugins        []string
		Replicas       int64
		Resources      *corev1.ResourceRequirements
		NodeSelector   map[string]string
		Roles          []v1alpha1.ElasticsearchClusterRole
		Version        string
	}{
		MinimumMasters: c.Spec.MinimumMasters,
		ESImage:        c.Spec.Image,
		PilotImage:     c.Spec.NavigatorClusterConfig.PilotImage,
		Sysctl:         c.Spec.NavigatorClusterConfig.Sysctls,
		Plugins:        c.Spec.Plugins,
		Replicas:       np.Replicas,
		Resources:      np.Resources,
		NodeSelector:   np.NodeSelector,
		Roles:          np.Roles,
		Version:        c.Spec.Version.String(),
	}

	hasher := fnv.New32a()
	hashutil.DeepHashObject(hasher, hashVar)

	// Add collisionCount in the hash if it exists.
	if collisionCount != nil {
		collisionCountBytes := make([]byte, 8)
		binary.LittleEndian.PutUint32(collisionCountBytes, uint32(*collisionCount))
		hasher.Write(collisionCountBytes)
	}

	return fmt.Sprintf("%d", hasher.Sum32())
}

func ClusterLabels(c *v1alpha1.ElasticsearchCluster) map[string]string {
	return map[string]string{
		"app":               "elasticsearch",
		ClusterNameLabelKey: c.Name,
	}
}

func NodePoolLabels(c *v1alpha1.ElasticsearchCluster, poolName string, roles ...v1alpha1.ElasticsearchClusterRole) map[string]string {
	labels := ClusterLabels(c)
	if poolName != "" {
		labels[NodePoolNameLabelKey] = poolName
	}
	for _, role := range roles {
		labels[string(role)] = "true"
	}
	return labels
}

func NodePoolResourceName(c *v1alpha1.ElasticsearchCluster, np *v1alpha1.ElasticsearchClusterNodePool) string {
	return fmt.Sprintf("%s-%s", ResourceBaseName(c), np.Name)
}

func SelectorForNodePool(c *v1alpha1.ElasticsearchCluster, np *v1alpha1.ElasticsearchClusterNodePool) (labels.Selector, error) {
	nodePoolNameReq, err := labels.NewRequirement(NodePoolNameLabelKey, selection.Equals, []string{np.Name})
	if err != nil {
		return nil, err
	}
	clusterSelector, err := SelectorForCluster(c)
	if err != nil {
		return nil, err
	}
	return clusterSelector.Add(*nodePoolNameReq), nil
}
