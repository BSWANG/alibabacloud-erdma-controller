package types

type Config struct {
	Region                             string            `json:"region"`
	ManageNonOwnedERIs                 bool              `json:"manageNonOwnedENIs"`
	ControllerNamespace                string            `json:"controllerNamespace"`
	ControllerName                     string            `json:"controllerName"`
	ClusterDomain                      string            `json:"clusterDomain"`
	CertDir                            string            `json:"certDir"`
	EnableDevicePlugin                 *bool             `json:"enableDevicePlugin"`
	EnableWebhook                      *bool             `json:"enableWebhook"`
	SMCInitImage                       string            `json:"smcInitImage"`
	EnableInitContainerInject          *bool             `json:"enableInitContainerInject"`
	NodeSelector                       map[string]string `json:"nodeSelector"`
	WaitNodeReadyTimeoutSeconds        int               `json:"waitNodeReadyTimeoutSeconds"`
	NodeMaxConcurrentReconciles        int               `json:"nodeMaxConcurrentReconciles"`
	ERdmaDeviceMaxConcurrentReconciles int               `json:"erdmaDeviceMaxConcurrentReconciles"`
	RateLimit                          map[string]int    `json:"rateLimit"`
	RateLimitBurst                     map[string]int    `json:"rateLimitBurst"`
}

type Sensitive string
type Credentials struct {
	Type            string    `json:"type"`
	AccessKeyID     Sensitive `json:"accessKeyID"`
	AccessKeySecret Sensitive `json:"accessKeySecret"`
	StsSecretNS     string    `json:"stsSecretNS"`
	StsSecretName   string    `json:"stsSecretName"`
}

func (c Sensitive) String() string {
	return "*******"
}
