package ci

import (
	"github.com/hofstadter-io/cuelm/schema"
)

Install: schema.#List & {
	items: [
		#Site.Ingress,
		#Site.Service,
		#Site.Deployment,
	]
}

Update: schema.#List & {
	items: [
		#Site.Deployment,
	]
}

#Site: {
	_Values: {
		name:      string | *"hof-docs" @tag(name)
		namespace: "websites"

		registry: "us.gcr.io/hof-io--develop"
		image:    string | *"\(domain)"
		version:  string | *"manual" @tag(version)

		domain: string | *"docs.hofstadter.io" @tag(domain)
		port:   80

		ga_mp_apikey: string | *"" @tag(ga_mp_apikey)

		#metadata: {
			name:      _Values.name
			namespace: _Values.namespace
			labels: {
				app: _Values.name
			}
			...
		}
	}

	Ingress: schema.#Ingress & {
		metadata: _Values.#metadata & {
			annotations: {
				"kubernetes.io/tls-acme":                         "true"
				"kubernetes.io/ingress.class":                    "nginx"
				"nginx.ingress.kubernetes.io/force-ssl-redirect": "true"
				"cert-manager.io/cluster-issuer":                 "letsencrypt-prod"
				"cert-manager.io/issue-temporary-certificate":    "true"
				"acme.cert-manager.io/http01-edit-in-place":      "true"
			}
		} // END Ingress.metadata

		spec: {
			tls: [{
				hosts: [_Values.domain]
				secretName: "\(_Values.name)-tls"
			}]

			rules: [{
				host: _Values.domain
				http: paths: [{
					backend: {
						service: {
							name: Service.metadata.name
							port: "number": Service.spec.ports[0].port
						}
					}
				}]
			}]

		} // END Ingress.spec
	} // END Ingress

	Service: schema.#Service & {
		metadata: _Values.#metadata
		spec: {
			selector: _Values.#metadata.labels
			type:     "NodePort"
			ports: [{
				port:       _Values.port
				targetPort: _Values.port
			}]
		}
	}

	Deployment: schema.#Deployment & {
		metadata: _Values.#metadata
		spec: {
			selector: matchLabels: _Values.#metadata.labels

			template: {
				metadata: labels: _Values.#metadata.labels
				spec: {
					containers: [{
						name:            "website"
						image:           "\(_Values.registry)/\(_Values.image):\(_Values.version)"
						imagePullPolicy: "Always"
						env: [{
							name:  "GA_MP_APIKEY"
							value: _Values.ga_mp_apikey
						}]
						ports: [{
							containerPort: _Values.port
							protocol:      "TCP"
						}]
						readinessProbe: {
							httpGet: port: _Values.port
							initialDelaySeconds: 6
							failureThreshold:    3
							periodSeconds:       10
						}
						livenessProbe: {
							httpGet: port: _Values.port
							initialDelaySeconds: 6
							failureThreshold:    3
							periodSeconds:       10
						}
					}]
				}
			}
		}
	}

}
