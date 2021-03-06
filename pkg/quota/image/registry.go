// Package image implements evaluators of usage for imagestreams and images. They are supposed
// to be passed to resource quota controller and origin resource quota admission plugin.
package image

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	quota "k8s.io/kubernetes/pkg/quota/v1"
	"k8s.io/kubernetes/pkg/quota/v1/generic"

	imagev1 "github.com/openshift/api/image/v1"
	imagev1typedclient "github.com/openshift/client-go/image/clientset/versioned/typed/image/v1"
	imagev1informer "github.com/openshift/client-go/image/informers/externalversions/image/v1"
	"github.com/openshift/origin/pkg/api/legacy"
)

var legacyObjectCountAliases = map[schema.GroupVersionResource]corev1.ResourceName{
	imagev1.GroupVersion.WithResource("imagestreams"): imagev1.ResourceImageStreams,
}

// NewEvaluators returns the list of static evaluators that manage more than counts
func NewReplenishmentEvaluators(f quota.ListerForResourceFunc, isInformer imagev1informer.ImageStreamInformer, imageClient imagev1typedclient.ImageStreamTagsGetter) []quota.Evaluator {
	// these evaluators have special logic
	result := []quota.Evaluator{
		NewImageStreamTagEvaluator(isInformer.Lister(), imageClient),
		NewImageStreamImportEvaluator(isInformer.Lister()),
	}
	// these evaluators require an alias for backwards compatibility
	for gvr, alias := range legacyObjectCountAliases {
		result = append(result,
			generic.NewObjectCountEvaluator(gvr.GroupResource(), generic.ListResourceUsingListerFunc(f, gvr), alias))
	}
	return result
}

// NewImageQuotaRegistryForAdmission returns a registry for quota evaluation of OpenShift resources related to images in
// internal registry. It evaluates only image streams and related virtual resources that can cause a creation
// of new image stream objects.
// This is different that is used for reconciliation because admission has to check all forms of a resource (legacy and groupified), but
// reconciliation only has to check one.
func NewReplenishmentEvaluatorsForAdmission(isInformer imagev1informer.ImageStreamInformer, imageClient imagev1typedclient.ImageStreamTagsGetter) []quota.Evaluator {
	result := []quota.Evaluator{
		NewImageStreamTagEvaluator(isInformer.Lister(), imageClient),
		NewImageStreamImportEvaluator(isInformer.Lister()),
		&evaluatorForLegacyResource{
			Evaluator:           NewImageStreamTagEvaluator(isInformer.Lister(), imageClient),
			LegacyGroupResource: legacy.Resource("imagestreamtags"),
		},
		&evaluatorForLegacyResource{
			Evaluator:           NewImageStreamImportEvaluator(isInformer.Lister()),
			LegacyGroupResource: legacy.Resource("imagestreamimports"),
		},
	}
	// these evaluators require an alias for backwards compatibility
	for gvr, alias := range legacyObjectCountAliases {
		result = append(result,
			generic.NewObjectCountEvaluator(gvr.GroupResource(), generic.ListResourceUsingListerFunc(nil, gvr), alias))
	}
	// add the handling for the old resources
	result = append(result,
		generic.NewObjectCountEvaluator(
			legacy.Resource("imagestreams"),
			generic.ListResourceUsingListerFunc(nil, imagev1.GroupVersion.WithResource("imagestreams")),
			imagev1.ResourceImageStreams))

	return result
}

type evaluatorForLegacyResource struct {
	quota.Evaluator
	LegacyGroupResource schema.GroupResource
}

func (e *evaluatorForLegacyResource) GroupResource() schema.GroupResource {
	return e.LegacyGroupResource
}
