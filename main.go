package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"os"
	"strings"
)

// โครงสร้างข้อมูลสำหรับ JSON และ HTML
type PageData struct {
	Insights   map[string]string      `json:"insights"`
	Headers    map[string]interface{} `json:"headers"`
	Cookies    map[string]string      `json:"cookies"`
	Envs       map[string]string      `json:"envs"`
	LogoutURL  string                 `json:"logout_url,omitempty"`
	AccountURL string                 `json:"account_url,omitempty"`
	Token      string                 `json:"token,omitempty"`
	JWTClaims  interface{}            `json:"jwt_claims"`
}

const htmlTemplate = `
<!DOCTYPE html>
<html>
<head>
    <title>OIDC Debugger (Go)</title>
    <link href="https://cdnjs.cloudflare.com/ajax/libs/prism/1.29.0/themes/prism-tomorrow.min.css" rel="stylesheet" />
    <style>
        body { font-family: 'Segoe UI', sans-serif; padding: 20px; background: #f4f7f6; margin: 0; position: relative; }
        .top-nav { position: absolute; top: 15px; right: 20px; display: flex; gap: 8px; }
        .btn { text-decoration: none; padding: 6px 14px; border-radius: 4px; font-size: 0.8rem; font-weight: bold; color: white; transition: opacity 0.2s; }
        .btn-logout { background: #dc3545; }
        .btn-account { background: #007bff; }
        .btn:hover { opacity: 0.8; }
        
        .card { background: white; padding: 20px; margin-bottom: 20px; border-radius: 10px; box-shadow: 0 4px 6px rgba(0,0,0,0.05); }
        .insight-container { display: grid; grid-template-columns: repeat(auto-fit, minmax(180px, 1fr)); gap: 15px; margin-bottom: 20px; }
        .insight-box { background: #007bff; color: white; padding: 12px; border-radius: 8px; text-align: center; }
        .insight-box.node { background: #6f42c1; }
        .insight-label { font-size: 0.7rem; text-transform: uppercase; opacity: 0.8; }
        .insight-value { font-size: 0.95rem; font-weight: bold; word-break: break-all; }
        
        pre { max-height: 400px; border-radius: 8px !important; margin: 0 !important; }
        h1 { font-size: 1.5rem; margin-top: 0; color: #222; }
        h3 { border-bottom: 2px solid #eee; padding-bottom: 8px; font-size: 1rem; display: flex; justify-content: space-between; align-items: center; }
        .toggle-btn { background: #6c757d; color: white; border: none; padding: 4px 10px; border-radius: 4px; cursor: pointer; font-size: 0.7rem; }
        .jwt-card { border-left: 5px solid #ffc107; }
    </style>
</head>
<body>
    <div class="top-nav">
        <a href="{{ .AccountURL }}" class="btn btn-account" target="_blank">Manage Account</a>
        <a href="{{ .LogoutURL }}" class="btn btn-logout">Logout</a>
    </div>

    <h1>OIDC & System Debugger</h1>
    
    <div class="insight-container">
        <div class="insight-box"><div class="insight-label">Pod Name</div><div class="insight-value">{{ .Insights.POD_NAME }}</div></div>
        <div class="insight-box"><div class="insight-label">Pod IP</div><div class="insight-value">{{ .Insights.POD_IP }}</div></div>
        <div class="insight-box node"><div class="insight-label">Node Name</div><div class="insight-value">{{ .Insights.NODE_NAME }}</div></div>
        <div class="insight-box node"><div class="insight-label">Node IP</div><div class="insight-value">{{ .Insights.NODE_IP }}</div></div>
    </div>

    {{ if .JWTClaims }}
    <div class="card jwt-card">
        <h3>Decoded JWT (Access Token Payload)</h3>
        <pre><code class="language-json">{{ .JWTClaims | marshal }}</code></pre>
    </div>
    {{ end }}

    <div class="card">
        <h3>
            HTTP Request Headers
            <button class="toggle-btn" onclick="toggleHeaders()">Show All Headers</button>
        </h3>
        <pre><code id="header-code" class="language-json"></code></pre>
    </div>

    <div class="card">
        <h3>Cookies</h3>
        <pre><code class="language-json">{{ .Cookies | marshal }}</code></pre>
    </div>

    <div class="card">
        <h3>Environment Variables</h3>
        <pre><code class="language-json">{{ .Envs | marshal }}</code></pre>
    </div>

    <div class="card">
        <h3>API Tester</h3>
        <div style="margin-bottom: 10px;">
            <label style="display:inline-block; width:100px; font-weight:bold;">URL:</label>
            <input type="text" id="apiUrl" value="" style="width: calc(100% - 120px); padding: 5px; border: 1px solid #ccc; border-radius: 4px;">
            <script>
                // Auto-populate the API URL with the 'api.' subdomain
                document.getElementById('apiUrl').value = window.location.protocol + "//api." + window.location.host;
            </script>
        </div>
        <div style="margin-bottom: 10px;">
            <label style="display:inline-block; width:100px; font-weight:bold;">app-token:</label>
            <input type="text" id="appToken" value="{{ .Token }}" style="width: calc(100% - 120px); padding: 5px; border: 1px solid #ccc; border-radius: 4px;" placeholder="Auto-populated token">
        </div>
        <div style="margin-bottom: 10px;">
            <label style="display:inline-block; width:100px; font-weight:bold; vertical-align: top;">Payload:</label>
            <textarea id="apiPayload" style="width: calc(100% - 120px); height: 60px; padding: 5px; border: 1px solid #ccc; border-radius: 4px; font-family: monospace;">{"message": "hello"}</textarea>
        </div>
        <div style="margin-bottom: 15px; padding-left: 105px;">
            <button class="btn btn-account" onclick="testApi('GET')">Test GET</button>
            <button class="btn btn-account" style="background:#28a745;" onclick="testApi('POST')">Test POST</button>
            <button class="btn btn-account" style="background:#ffc107; color:black;" onclick="testApi('OPTIONS')">Test Preflight (OPTIONS)</button>
        </div>
        <pre><code id="apiResponse" class="language-json">Response will appear here...</code></pre>
    </div>

    <script src="https://cdnjs.cloudflare.com/ajax/libs/prism/1.29.0/components/prism-core.min.js"></script>
    <script src="https://cdnjs.cloudflare.com/ajax/libs/prism/1.29.0/components/prism-json.min.js"></script>
    
    <script>
        const allHeaders = {{ .Headers | marshal_raw }};
        const hideList = [
            "Accept", "Accept-Encoding", "Accept-Language", "Cache-Control", 
            "Dnt", "Priority", "Sec-Ch-Ua", "Sec-Ch-Ua-Mobile", 
            "Sec-Ch-Ua-Platform", "Sec-Fetch-Dest", "Sec-Fetch-Mode", 
            "Sec-Fetch-Site", "Sec-Fetch-User", "Upgrade-Insecure-Requests", "User-Agent"
        ];
        
        let showFull = false;

        function renderHeaders() {
            let displayHeaders = {};
            if (showFull) {
                displayHeaders = allHeaders;
            } else {
                for (let key in allHeaders) {
                    if (!hideList.includes(key)) {
                        displayHeaders[key] = allHeaders[key];
                    }
                }
            }
            const codeElem = document.getElementById('header-code');
            codeElem.textContent = JSON.stringify(displayHeaders, null, 2);
            Prism.highlightElement(codeElem);
        }

        function toggleHeaders() {
            showFull = !showFull;
            document.querySelector('.toggle-btn').textContent = showFull ? "Hide Common Headers" : "Show All Headers";
            renderHeaders();
        }

        async function testApi(method) {
            const url = document.getElementById('apiUrl').value;
            const token = document.getElementById('appToken').value;
            const payload = document.getElementById('apiPayload').value;
            const codeElem = document.getElementById('apiResponse');
            
            codeElem.textContent = "Loading Preflight (OPTIONS)...";
            let output = {};
            
            try {
                if (method !== 'OPTIONS') {
                    // --- 1. จำลองการส่ง Preflight (OPTIONS) แบบแยกให้เห็นชัดเจน ---
                    const preflightOpts = {
                        method: 'OPTIONS',
                        headers: {
                            'Access-Control-Request-Method': method,
                            'Access-Control-Request-Headers': 'app-token, content-type'
                        }
                    };
                    const preflightRes = await fetch(url, preflightOpts);
                    
                    output.preflight_options = {
                        status: preflightRes.status,
                        statusText: preflightRes.statusText,
                        headers: {}
                    };
                    
                    preflightRes.headers.forEach((val, key) => {
                         output.preflight_options.headers[key] = val;
                    });
                    
                    codeElem.textContent = "Preflight done. Loading " + method + "...";
                }

                // --- 2. ส่ง Request จริง (GET / POST) ---
                const options = {
                    method: method,
                    headers: {
                        'app-token': token,
                        'Content-Type': 'application/json'
                    }
                };
                
                if (method !== 'GET' && method !== 'HEAD' && method !== 'OPTIONS') {
                    options.body = payload;
                }
                
                const response = await fetch(url, options);
                const text = await response.text();
                let data;
                try {
                    data = JSON.parse(text);
                } catch(e) {
                    data = text;
                }
                
                output.actual_request = {
                    method: method,
                    status: response.status,
                    statusText: response.statusText,
                    data: data
                };
                
                codeElem.textContent = JSON.stringify(output, null, 2);
                Prism.highlightElement(codeElem);

            } catch(error) {
                codeElem.textContent = "Error: " + error.message;
            }
        }

        renderHeaders();
    </script>
</body>
</html>
`

func decodeJWT(token string) interface{} {
	parts := strings.Split(token, ".")
	if len(parts) < 2 { return nil }
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil { return nil }
	var result interface{}
	json.Unmarshal(payload, &result)
	return result
}

func main() {
	funcMap := template.FuncMap{
		"marshal": func(v interface{}) string {
			a, _ := json.MarshalIndent(v, "", "  ")
			return string(a)
		},
		"marshal_raw": func(v interface{}) template.JS {
			a, _ := json.Marshal(v)
			return template.JS(a)
		},
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// เตรียมข้อมูลพื้นฐาน
		keycloakURL := os.Getenv("KEYCLOAK_URL")
		realm := os.Getenv("REALM")
		clientID := os.Getenv("CLIENT_ID")
		
		scheme := "http"
		if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" { scheme = "https" }
		baseURL := fmt.Sprintf("%s://%s", scheme, r.Host)

		keycloakLogout := fmt.Sprintf("%s/realms/%s/protocol/openid-connect/logout?client_id=%s&post_logout_redirect_uri=%s/",
			keycloakURL, realm, clientID, baseURL)
		
		data := PageData{
			Insights: map[string]string{
				"POD_NAME":  os.Getenv("POD_NAME"),
				"POD_IP":    os.Getenv("POD_IP"),
				"NODE_NAME": os.Getenv("NODE_NAME"),
				"NODE_IP":   os.Getenv("NODE_IP"),
			},
			Headers:    make(map[string]interface{}),
			Cookies:    make(map[string]string),
			Envs:       make(map[string]string),
			LogoutURL:  fmt.Sprintf("%s/oauth2/sign_out?rd=%s", baseURL, url.QueryEscape(keycloakLogout)),
			AccountURL: fmt.Sprintf("%s/realms/%s/account", keycloakURL, realm),
		}

		// รวบรวม Headers และถอดรหัส JWT
		for k, v := range r.Header {
			if len(v) > 0 {
				data.Headers[k] = v[0]
				if k == "X-Auth-Request-Access-Token" || k == "Authorization" {
					token := v[0]
					if strings.HasPrefix(token, "Bearer ") {
						token = strings.TrimPrefix(token, "Bearer ")
					}
					data.JWTClaims = decodeJWT(token)
					data.Token = token
				}
			}
		}

		for _, c := range r.Cookies() { data.Cookies[c.Name] = c.Value }
		
		// --- เพิ่มส่วนจัดการ CORS & Preflight ---
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "app-token, Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		// --- ส่วนที่เพิ่ม: ตรวจสอบ Method ---
		if r.Method == http.MethodPost {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(data)
			return
		}

		// ถ้าเป็น GET ให้แสดง HTML เหมือนเดิม
		tmpl := template.Must(template.New("web").Funcs(funcMap).Parse(htmlTemplate))
		tmpl.Execute(w, data)
	})

	port := os.Getenv("PORT")
	if port == "" { port = "5000" }
	fmt.Printf("Server starting on :%s\n", port)
	http.ListenAndServe(":"+port, nil)
}