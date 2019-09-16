package webhook

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"gopkg.in/yaml.v2"
	"k8s.io/api/admission/v1beta1"
	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/klog"
)

const (
	DefaultConfigFile                   = ""                              // default config file
	DefaultAnnotation                   = "sidecar-injector.lxcfs/inject" // default annotation
	DefaultNamespace                    = "default"                       // default namespace
	DefaultAnnotationRequired           = true                            // annotation is required
	admissionWebhookAnnotationStatusKey = "sidecar-injector.lxcfs/status"
)

type WebhookServer struct {
	Config *Config
	Server *http.Server
}

type Config struct {
	ConfigFile        string         `yaml:"configFile"`
	Annotation        string         `yaml:"annotation"`
	AnnotationRequied bool           `yaml:"requireAnnotation"`
	Namespace         string         `yaml:"namespace"`
	SidecarConfig     *SidecarConfig `yaml:"sidecarConfig"`
}

type SidecarConfig struct {
	VolumeMounts []corev1.VolumeMount `yaml:"volumeMounts"`
	Volumes      []corev1.Volume      `yaml:"volumes"`
}

type patchOperation struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value,omitempty"`
}

var (
	runtimeScheme = runtime.NewScheme()
	codecs        = serializer.NewCodecFactory(runtimeScheme)
	deserializer  = codecs.UniversalDeserializer()
)

func init() {
	addToScheme(runtimeScheme)
}

func addToScheme(scheme *runtime.Scheme) {
	utilruntime.Must(corev1.AddToScheme(scheme))
	utilruntime.Must(admissionregistrationv1beta1.AddToScheme(scheme))
	utilruntime.Must(v1beta1.AddToScheme(scheme))
}

func LoadWebhookServerConfig(cfgFile string) (*Config, error) {
	var cfg *Config
	if len(cfgFile) != 0 {
		cfg.ConfigFile = cfgFile
		data, err := ioutil.ReadFile(cfgFile)
		if err != nil {
			return nil, err
		}

		err = yaml.Unmarshal(data, cfg)
		if err != nil {
			return nil, err
		}
	} else {
		klog.Info("Failed to load sidecar config file, fallback to default...")
		cfg = &Config{
			Annotation:        DefaultAnnotation,
			AnnotationRequied: DefaultAnnotationRequired,
			Namespace:         DefaultNamespace,
		}
	}

	cfg.SidecarConfig = &SidecarConfig{
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      "lxcfs-proc-cpuinfo",
				MountPath: "/proc/cpuinfo",
			},
			{
				Name:      "lxcfs-proc-meminfo",
				MountPath: "/proc/meminfo",
			},
			{
				Name:      "lxcfs-proc-diskstats",
				MountPath: "/proc/diskstats",
			},
			{
				Name:      "lxcfs-proc-stat",
				MountPath: "/proc/stat",
			},
			{
				Name:      "lxcfs-proc-swaps",
				MountPath: "/proc/swaps",
			},
			{
				Name:      "lxcfs-proc-uptime",
				MountPath: "/proc/uptime",
			},
		},
		Volumes: []corev1.Volume{
			{
				Name: "lxcfs-proc-cpuinfo",
				VolumeSource: corev1.VolumeSource{
					HostPath: &corev1.HostPathVolumeSource{
						Path: "/var/lib/lxcfs/proc/cpuinfo",
					},
				},
			},
			{
				Name: "lxcfs-proc-diskstats",
				VolumeSource: corev1.VolumeSource{
					HostPath: &corev1.HostPathVolumeSource{
						Path: "/var/lib/lxcfs/proc/diskstats",
					},
				},
			},
			{
				Name: "lxcfs-proc-meminfo",
				VolumeSource: corev1.VolumeSource{
					HostPath: &corev1.HostPathVolumeSource{
						Path: "/var/lib/lxcfs/proc/meminfo",
					},
				},
			},
			{
				Name: "lxcfs-proc-stat",
				VolumeSource: corev1.VolumeSource{
					HostPath: &corev1.HostPathVolumeSource{
						Path: "/var/lib/lxcfs/proc/stat",
					},
				},
			},
			{
				Name: "lxcfs-proc-swaps",
				VolumeSource: corev1.VolumeSource{
					HostPath: &corev1.HostPathVolumeSource{
						Path: "/var/lib/lxcfs/proc/swaps",
					},
				},
			},
			{
				Name: "lxcfs-proc-uptime",
				VolumeSource: corev1.VolumeSource{
					HostPath: &corev1.HostPathVolumeSource{
						Path: "/var/lib/lxcfs/proc/uptime",
					},
				},
			},
		},
	}

	return cfg, nil
}

// Mutate method for webhook server
func (whsvr *WebhookServer) Mutate(w http.ResponseWriter, r *http.Request) {
	var body []byte
	if r.Body != nil {
		if data, err := ioutil.ReadAll(r.Body); err == nil {
			body = data
		}
	}
	if len(body) == 0 {
		klog.Error("empty body")
		http.Error(w, "empty body", http.StatusBadRequest)
		return
	}

	// verify the content type
	contentType := r.Header.Get("Content-Type")
	if contentType != "application/json" {
		klog.Errorf("Content-Type=%s, expect application/json", contentType)
		http.Error(w, "invalid Content-Type, expect `application/json`", http.StatusUnsupportedMediaType)
		return
	}

	klog.V(2).Info("handling request: %s", body)

	// The AdmissionReview that was sent to the webhook
	requestedAdmissionReview := &v1beta1.AdmissionReview{}

	var admissionResponse *v1beta1.AdmissionResponse

	if _, _, err := deserializer.Decode(body, nil, requestedAdmissionReview); err != nil {
		klog.Errorf("Can't decode body: %v", err)
		admissionResponse = toAdmissionResponseErr(err)
	} else {
		// pass to admitFunc
		admissionResponse = whsvr.mutate(requestedAdmissionReview)
	}

	// The AdmissionReview that will be returned
	responseAdmissionReview := &v1beta1.AdmissionReview{}
	if admissionResponse != nil {
		responseAdmissionReview.Response = admissionResponse
		if requestedAdmissionReview.Request != nil {
			responseAdmissionReview.Response.UID = requestedAdmissionReview.Request.UID
		}
	}

	klog.V(2).Info(fmt.Sprintf("sending response: %v", responseAdmissionReview.Response))

	resp, err := json.Marshal(responseAdmissionReview)
	if err != nil {
		klog.Errorf("Can't encode response: %v", err)
		http.Error(w, fmt.Sprintf("could not encode response: %v", err), http.StatusInternalServerError)
	}
	if _, err := w.Write(resp); err != nil {
		klog.Errorf("Can't write response: %v", err)
		http.Error(w, fmt.Sprintf("could not encode response: %v", err), http.StatusInternalServerError)
	}
}

func (whsvr *WebhookServer) mutate(ar *v1beta1.AdmissionReview) *v1beta1.AdmissionResponse {
	req := ar.Request
	var pod corev1.Pod
	if err := json.Unmarshal(req.Object.Raw, &pod); err != nil {
		klog.Errorf("Could not unmarshal raw object: %v", err)
		return toAdmissionResponseErr(err)
	}

	klog.Infof("AdmissionReview for Kind=%v, Namespace=%v Name=%v (%v) UID=%v patchOperation=%v UserInfo=%v",
		req.Kind, req.Namespace, req.Name, pod.Name, req.UID, req.Operation, req.UserInfo)

	if !whsvr.mutationRequired(&pod.ObjectMeta) {
		klog.Info("Skipping mutation")
		return &v1beta1.AdmissionResponse{
			Allowed: true,
		}
	}

	klog.Info("Mutating pod")
	annotations := map[string]string{admissionWebhookAnnotationStatusKey: "injected"}
	patchBytes, err := createPatch(&pod, whsvr.Config.SidecarConfig, annotations)
	if err != nil {
		klog.Errorf("Could not create patch: %v", err)
		return toAdmissionResponseErr(err)
	}

	return &v1beta1.AdmissionResponse{
		Allowed: true,
		Patch:   patchBytes,
		PatchType: func() *v1beta1.PatchType {
			pt := v1beta1.PatchTypeJSONPatch
			return &pt
		}(),
	}
}

func (whsvr *WebhookServer) mutationRequired(metadata *metav1.ObjectMeta) bool {
	if !whsvr.Config.AnnotationRequied {
		return false
	}

	annotations := metadata.GetAnnotations()
	if annotations == nil {
		return false
	}

	var required bool
	switch annotations[whsvr.Config.Annotation] {
	case "yes", "true":
		required = true
	default:
		required = false
	}

	return required
}

func addVolumeMount(target []corev1.Container, added []corev1.VolumeMount, basePath string) (patch []patchOperation) {
	for i, c := range target {
		target[i].VolumeMounts = append(c.VolumeMounts, added...)
	}

	return append(patch, patchOperation{
		Op:    "replace",
		Path:  basePath,
		Value: target,
	})
}

func addVolume(target, added []corev1.Volume, basePath string) (patch []patchOperation) {
	first := len(target) == 0
	var value interface{}
	for _, add := range added {
		value = add
		path := basePath
		if first {
			first = false
			value = []corev1.Volume{add}
		} else {
			path += "/-"
		}
		patch = append(patch, patchOperation{
			Op:    "add",
			Path:  path,
			Value: value,
		})
	}
	return patch
}

func updateAnnotation(target map[string]string, added map[string]string) (patch []patchOperation) {
	for key, value := range added {
		if target == nil || target[key] == "" {
			target = map[string]string{}
			patch = append(patch, patchOperation{
				Op:   "add",
				Path: "/metadata/annotations",
				Value: map[string]string{
					key: value,
				},
			})
		} else {
			patch = append(patch, patchOperation{
				Op:    "replace",
				Path:  "/metadata/annotations/" + key,
				Value: value,
			})
		}
	}
	return patch
}

// create mutation patch for resoures
func createPatch(pod *corev1.Pod, cfg *SidecarConfig, annotations map[string]string) ([]byte, error) {
	var patch []patchOperation

	// add volume mounts
	patch = append(patch, addVolumeMount(pod.Spec.Containers, cfg.VolumeMounts, "/spec/containers")...)
	// add volumes
	patch = append(patch, addVolume(pod.Spec.Volumes, cfg.Volumes, "/spec/volumes")...)
	// update annotations
	patch = append(patch, updateAnnotation(pod.Annotations, annotations)...)

	return json.Marshal(patch)
}

// toAdmissionResponseErr() is a helper function to create an AdmissionResponse
// with an embedded error
func toAdmissionResponseErr(err error) *v1beta1.AdmissionResponse {
	return &v1beta1.AdmissionResponse{
		Result: &metav1.Status{
			Message: err.Error(),
		},
	}
}
