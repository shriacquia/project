package controllers

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	cronjobv1 "tutorial.kubebuilder.io/project/api/v1"
)

/*
The first step to writing a simple integration test is to actuall create an instance of CronJob you
can run tests agains. Note that to create a CronJob, you'll need to create a stub CronJob struct
that contains your CronJob's specifications.
Note that when we create a stub CronJob, the CronJob also needs stubs required of its downstream
objects. Without the stubbed JobTemplateSpec and the PodTemplateSpec below, the kube API will not
be able to create the CronJob
*/

var _ = Describe("CronJob controller", func() {
	// Define utility constants for object names and testing timeouts/durations and intervals
	const (
		CronjobName      = "test-cronjob"
		CronJobNamespace = "default"
		JobName          = "test-job"

		timeout  = time.Second * 10
		duration = time.Second * 10
		interval = time.Millisecond * 250
	)

	Context("When updating CronJob Status", func() {
		It("Should increase the CronJob Status.Active count when new jobs are created", func() {
			By("Creating a new CronJob")
			ctx := context.Background()
			cronJob := &cronjobv1.CronJob{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "batch.tutorial.kubebuilder.io/v1",
					Kind:       "CronJob",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      CronjobName,
					Namespace: CronJobNamespace,
				},
				Spec: cronjobv1.CronJobSpec{
					Schedule: "1 * * * *",
					JobTemplate: batchv1.JobTemplateSpec{
						Spec: batchv1.JobSpec{
							// For simplicity we only fill out the required fields
							Template: v1.PodTemplateSpec{
								Spec: v1.PodSpec{
									Containers: []v1.Container{
										{
											Name:  "test-container",
											Image: "test-image",
										},
									},
									RestartPolicy: v1.RestartPolicyOnFailure,
								},
							},
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, cronJob)).Should(Succeed())

			/*
				After creating this CronJob, let's check that the CronJob's Spec fields match what we passed in.
				Note that, because the k8s apiserver may not have finished creating a CronJob after our
				Create() call from earlier, we will use Gomega's `Eventually()` testing function instead of
				`Expect()` to give the apiserver an opportunity to finish creating our CronJob.
				`Eventually()` will repeatedly run the function provided as an argument every interval seconds
				until
					(a) the function's output matches what's expected in the subsequent `Should()` call or
					(b) the number of attempts * interval period exceed the provided timeout value.

				In the examples below, timeout and interval are Go Duration values of our choosing.
			*/

			cronjobLookupKey := types.NamespacedName{Name: CronjobName, Namespace: CronJobNamespace}
			createdCronjob := &cronjobv1.CronJob{}

			// We will have to retry getting this newly created CronJob, given that creation may not happen immediately.
			Eventually(func() bool {
				err := k8sClient.Get(ctx, cronjobLookupKey, createdCronjob)
				return err == nil
			}, timeout, interval).Should(BeTrue())
			// Let's make sure our Schedule string value was properly converted/handled.
			Expect(createdCronjob.Spec.Schedule).Should(Equal("1 * * * *"))

		})
	})

})
