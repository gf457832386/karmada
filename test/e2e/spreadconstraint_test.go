package e2e

import (
	"context"
	"fmt"

	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"

	clusterv1alpha1 "github.com/karmada-io/karmada/pkg/apis/cluster/v1alpha1"
	policyv1alpha1 "github.com/karmada-io/karmada/pkg/apis/policy/v1alpha1"
	"github.com/karmada-io/karmada/test/e2e/framework"
	"github.com/karmada-io/karmada/test/helper"
)

var _ = ginkgo.Describe("propagation with spreadConstraint testing", func() {
	ginkgo.Context("spreadConstraint testing", func() {
		klog.Infof("testtt1")
		policyNamespace := testNamespace
		policyName := deploymentNamePrefix + rand.String(RandomStrLength)
		deploymentNamespace := testNamespace
		deploymentName := policyName
		deployment := helper.NewDeployment(deploymentNamespace, deploymentName) //生成一个deployment

		originalClusterProviderInfo := make(map[string]string)
		originalClusterRegionInfo := make(map[string]string)
		//desiredSpreadFieldValue := "provider"
		//undesiredRegion := []string{"cn-north-1"}
		desiredScheduleResult := "member1"

		// desire to schedule to clusters having spreadConstraint provider/region/zone property
		spreadConstraints := []policyv1alpha1.SpreadConstraint{
			{
				SpreadByField: policyv1alpha1.SpreadByFieldCluster,
				//SpreadByField: policyv1alpha1.SpreadFieldValue("provider"),
				//SpreadByField: policyv1alpha1.SpreadByFieldCluster,
				//SpreadByField: policyv1alpha1.SpreadByFieldCluster,
			},
			//{
			//SpreadByField: policyv1alpha1.SpreadFieldValue("region"),
			//},
		}

		policy := helper.NewPropagationPolicy(policyNamespace, policyName, []policyv1alpha1.ResourceSelector{
			{
				APIVersion: deployment.APIVersion,
				Kind:       deployment.Kind,
				Name:       deployment.Name,
			},
		}, policyv1alpha1.Placement{
			ClusterAffinity: &policyv1alpha1.ClusterAffinity{
				ClusterNames: framework.ClusterNames(),
			},
			SpreadConstraints: spreadConstraints,
		})

		ginkgo.BeforeEach(func() { //给各个集群赋值属性
			ginkgo.By("setting provider and region for clusters", func() {
				providerMap := []string{"huaweicloud", "huaweicloud", ""}
				regionMap := []string{"cn-south-1", "", "cn-east-1"}
				for index, cluster := range framework.ClusterNames() {
					if index > 2 {
						break
					}
					fmt.Printf("setting provider and region for cluster %v\n", cluster)
					gomega.Eventually(func(g gomega.Gomega) {
						clusterObj := &clusterv1alpha1.Cluster{}
						err := controlPlaneClient.Get(context.TODO(), client.ObjectKey{Name: cluster}, clusterObj)
						g.Expect(err).NotTo(gomega.HaveOccurred())
						klog.Infof("testtt2")
						originalClusterProviderInfo[cluster] = clusterObj.Spec.Provider
						originalClusterRegionInfo[cluster] = clusterObj.Spec.Region
						clusterObj.Spec.Provider = providerMap[index]
						clusterObj.Spec.Region = regionMap[index]
						klog.Infof("testtt2(%s)", clusterObj.Spec.Provider)
						klog.Infof("testtt2(%s)", clusterObj.Spec.Region)
						err = controlPlaneClient.Update(context.TODO(), clusterObj)
						g.Expect(err).NotTo(gomega.HaveOccurred())
					}, pollTimeout, pollInterval).Should(gomega.Succeed())
				}
			})
		})

		ginkgo.AfterEach(func() {
			ginkgo.By("recovering provider and region for clusters", func() {
				for index, cluster := range framework.ClusterNames() {
					if index > 2 {
						break
					}
					klog.Infof("testtt3")
					gomega.Eventually(func(g gomega.Gomega) {
						clusterObj := &clusterv1alpha1.Cluster{}
						err := controlPlaneClient.Get(context.TODO(), client.ObjectKey{Name: cluster}, clusterObj)
						g.Expect(err).NotTo(gomega.HaveOccurred())

						clusterObj.Spec.Provider = originalClusterProviderInfo[cluster]
						clusterObj.Spec.Region = originalClusterRegionInfo[cluster]
						err = controlPlaneClient.Update(context.TODO(), clusterObj)
						g.Expect(err).NotTo(gomega.HaveOccurred())
					}, pollTimeout, pollInterval).Should(gomega.Succeed())
				}
			})
		})

		ginkgo.It("propagation with spreadConstraint testing", func() {
			framework.CreatePropagationPolicy(karmadaClient, policy)
			framework.CreateDeployment(kubeClient, deployment)

			ginkgo.By("check whether deployment is scheduled to clusters which meeting the spreadConstraint requirements", func() {

				targetClusterNames := framework.ExtractTargetClustersFrom(controlPlaneClient, deployment)
				klog.Infof("length(%s)", len(targetClusterNames))
				klog.Infof("targetClusterNames(%s)", targetClusterNames)
				klog.Infof("testtt4")
				gomega.Expect(len(targetClusterNames) == 2).Should(gomega.BeTrue())
				klog.Infof("testtt5")
				gomega.Expect(targetClusterNames[0] == desiredScheduleResult).Should(gomega.BeTrue())

			})

			framework.RemoveDeployment(kubeClient, deployment.Namespace, deployment.Name)
			framework.RemovePropagationPolicy(karmadaClient, policy.Namespace, policy.Name)
		})
	})
})
