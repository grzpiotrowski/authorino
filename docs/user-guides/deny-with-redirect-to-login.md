# User guide: Redirecting to a login page

Customize response status code and headers on failed requests to redirect users of a web application protected with Authorino to a login page instead of a `401 Unauthorized`.

<details>
  <summary>
    <strong>Authorino features in this guide:</strong>
    <ul>
      <li>Dynamic response → <a href="./../features.md#extra-custom-denial-status-denywith">Custom denial status</a></li>
      <li>Identity verification & authentication → <a href="./../features.md#api-key-identityapikey">API key</a></li>
      <li>Identity verification & authentication → <a href="./../features.md#openid-connect-oidc-jwtjose-verification-and-validation-identityoidc">OpenID Connect (OIDC) JWT/JOSE verification and validation</a></li>
    </ul>
  </summary>

  Authorino's default response status codes, messages and headers for unauthenticated (`401`) and unauthorized (`403`) requests can be customized with static values and values fetched from the [Authorization JSON](./../architecture.md#the-authorization-json).

  Check out as well the user guides about [HTTP "Basic" Authentication (RFC 7235)](./user-guides/http-basic-authentication.md) and [OpenID Connect Discovery and authentication with JWTs](./oidc-jwt-authentication.md).

  For further details about Authorino features in general, check the [docs](./../features.md).
</details>

<br/>

## Requirements

- Kubernetes server

Create a containerized Kubernetes server locally using [Kind](https://kind.sigs.k8s.io):

```sh
kind create cluster --name authorino-trial
```

## 1. Install the Authorino Operator

```sh
git clone https://github.com/kuadrant/authorino-operator && cd authorino-operator
kubectl create namespace authorino-operator && make install deploy
```

## 2. Create the namespace

```sh
kubectl create namespace authorino
```

## 3. Deploy the Matrix Quotes web application

The **Matrix Quotes** is a static web application that contains quotes from the film _The Matrix_.

```sh
kubectl -n authorino apply -f https://raw.githubusercontent.com/kuadrant/authorino-examples/main/matrix-quotes/matrix-quotes-deploy.yaml
```

## 4. Deploy Authorino

```sh
kubectl -n authorino apply -f -<<EOF
apiVersion: operator.authorino.kuadrant.io/v1beta1
kind: Authorino
metadata:
  name: authorino
spec:
  listener:
    tls:
      enabled: false
  oidcServer:
    tls:
      enabled: false
EOF
```

The command above will deploy Authorino as a separate service (as oposed to a sidecar of the protected API and other architectures), in `namespaced` reconciliation mode, and with TLS termination disabled. For other variants and deployment options, check out the [Getting Started](./../getting-started.md#2-deploy-an-authorino-instance) section of the docs, the [Architecture](./../architecture.md#topologies) page, and the spec for the [`Authorino`](https://github.com/Kuadrant/authorino-operator/blob/main/config/crd/bases/operator.authorino.kuadrant.io_authorinos.yaml) CRD in the Authorino Operator repo.

## 5. Setup Envoy

The following bundle from the Authorino examples (manifest referred in the command below) is to apply Envoy configuration and deploy Envoy proxy, that wire up the Matrix Quotes webapp behind the reverse-proxy and external authorization with the Authorino instance.

For details and instructions to setup Envoy manually, see _Protect a service > Setup Envoy_ in the [Getting Started](./../getting-started.md#1-setup-envoy) page. For a simpler and straighforward way to manage an API, without having to manually install or configure Envoy and Authorino, check out [Kuadrant](https://github.com/kuadrant).

```sh
kubectl -n authorino apply -f https://raw.githubusercontent.com/kuadrant/authorino-examples/main/matrix-quotes/envoy-deploy.yaml
```

The bundle also creates an `Ingress` with host name `matrix-quotes-authorino.127.0.0.1.nip.io`, but if you are using a local Kubernetes cluster created with Kind, you need to forward requests on port 8000 to inside the cluster in order to actually reach the Envoy service:

```sh
kubectl -n authorino port-forward deployment/envoy 8000:8000 &
```

## 6. Create the `AuthConfig`

```sh
kubectl -n authorino apply -f -<<EOF
apiVersion: authorino.3scale.net/v1beta1
kind: AuthConfig
metadata:
  name: matrix-quotes-protection
spec:
  hosts:
  - matrix-quotes-authorino.127.0.0.1.nip.io
  identity:
  - name: browser-users
    apiKey:
      labelSelectors:
        group: users
    credentials:
      in: cookie
      keySelector: TOKEN
  - name: http-basic-auth
    apiKey:
      labelSelectors:
        group: users
    credentials:
      in: authorization_header
      keySelector: Basic
  denyWith:
    unauthenticated:
      code: 302
      headers:
      - name: Location
        valueFrom:
          authJSON: http://matrix-quotes-authorino.127.0.0.1.nip.io:8000/login.html?redirect_to={context.request.http.path}
EOF
```

Check out the docs for information about the common feature [JSON paths](./../features.md#common-feature-json-paths-valuefromauthjson) for reading from the [Authorization JSON](./../architecture.md#the-authorization-json).

## 7. Create an API key

```sh
kubectl -n authorino apply -f -<<EOF
apiVersion: v1
kind: Secret
metadata:
  name: user-credential-1
  labels:
    authorino.3scale.net/managed-by: authorino
    group: users
stringData:
  api_key: am9objpw # john:p
type: Opaque
EOF
```

## 8. Consume the application

On a web browser, navigate to http://matrix-quotes-authorino.127.0.0.1.nip.io:8000.

Click on the cards to read quotes from characters of the movie. You should be redirected to login page.

Log in using John's credentials:
- **Username:** john
- **Password:** p

Click again on the cards and check that now you are able to access the inner pages.

You can also consume a protected endpoint of the application using HTTP Basic Authentication:

```sh
curl -u john:p http://matrix-quotes-authorino.127.0.0.1.nip.io:8000/neo.html
# HTTP/1.1 200 OK
```

## 9. (Optional) Modify the `AuthConfig` to authenticate with OIDC

### Setup a Keycloak server

Deploy a Keycloak server preloaded with a realm named `kuadrant`:

```sh
kubectl -n authorino apply -f https://raw.githubusercontent.com/kuadrant/authorino-examples/main/keycloak/keycloak-deploy.yaml
```

Resolve local Keycloak domain so it can be accessed from the local host and inside the cluster with the name: (This will be needed to redirect to Keycloak's login page and at the same time validate issued tokens.)

```sh
echo '127.0.0.1 keycloak' >> /etc/hosts
```

Forward local requests to the instance of Keycloak running in the cluster:

```sh
kubectl -n authorino port-forward deployment/keycloak 8080:8080 &
```

Create a client:

```sh
curl -H "Authorization: Bearer $(curl http://keycloak:8080/auth/realms/master/protocol/openid-connect/token -s -d 'grant_type=password' -d 'client_id=admin-cli' -d 'username=admin' -d 'password=p' | jq -r .access_token)" \
     -H 'Content-type: application/json' \
     -d '{ "name": "matrix-quotes", "clientId": "matrix-quotes", "publicClient": true, "redirectUris": ["http://matrix-quotes-authorino.127.0.0.1.nip.io:8000/auth*"], "enabled": true }' \
     http://keycloak:8080/auth/admin/realms/kuadrant/clients
```

### Reconfigure the Matrix Quotes app to use Keycloak's login page

```sh
kubectl -n authorino set env deployment/matrix-quotes KEYCLOAK_REALM=http://keycloak:8080/auth/realms/kuadrant CLIENT_ID=matrix-quotes
```

### Apply the changes to the `AuthConfig`

```sh
kubectl -n authorino apply -f -<<EOF
apiVersion: authorino.3scale.net/v1beta1
kind: AuthConfig
metadata:
  name: matrix-quotes-protection
spec:
  hosts:
  - matrix-quotes-authorino.127.0.0.1.nip.io
  identity:
  - name: idp-users
    oidc:
      endpoint: http://keycloak:8080/auth/realms/kuadrant
    credentials:
      in: cookie
      keySelector: TOKEN
  denyWith:
    unauthenticated:
      code: 302
      headers:
      - name: Location
        valueFrom:
          authJSON: http://keycloak:8080/auth/realms/kuadrant/protocol/openid-connect/auth?client_id=matrix-quotes&redirect_uri=http://matrix-quotes-authorino.127.0.0.1.nip.io:8000/auth?redirect_to={context.request.http.path}&scope=openid&response_type=code
EOF
```

### Consume the application again

Refresh the browser window or navigate again to http://matrix-quotes-authorino.127.0.0.1.nip.io:8000.

Click on the cards to read quotes from characters of the movie. You should be redirected to login page this time served by the Keycloak server.

Log in as Jane (a user of the Keycloak realm):
- **Username:** jane
- **Password:** p

Click again on the cards and check that now you are able to access the inner pages.

## Cleanup

If you have started a Kubernetes cluster locally with Kind only to test this user guide, delete it by running:

```sh
kind delete cluster --name authorino-trial
```

Otherwise, delete the namespaces created in step 1 and 2:

```sh
kubectl -n authorino namespace authorino
kubectl -n authorino namespace authorino-operator
```

To uninstall the Authorino and Authorino Operator manifests, run from the Authorino Operator directory:

```sh
make uninstall
```