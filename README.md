# OIDC Debugger

แอป Go สำหรับตรวจสอบข้อมูล request ที่ส่งผ่าน reverse proxy หรือระบบ SSO มาถึง backend แสดง headers, cookies, JWT payload และข้อมูล Pod/Node พร้อม API tester บนหน้าเว็บ เหมาะสำหรับ debug การเชื่อมต่อ OAuth2 Proxy, Keycloak และ API gateway

แอปนี้เป็น request inspector ไม่ได้เริ่ม OIDC authorization flow หรือแลก authorization code เป็น token เอง และไม่ใช่โปรเจกต์เว็บไซต์ oidcdebugger.com

## เริ่มใช้งานด้วย Docker

```sh
git clone git@github.com:Thirachart/oidc-debugger.git
cd oidc-debugger
docker build -t oidc-debugger .
docker run --rm --name oidc-debugger -p 127.0.0.1:5000:5000 oidc-debugger
```

เปิด <http://localhost:5000> เพื่อดู request ปัจจุบัน การรันโดยไม่มี proxy ยังใช้ตรวจ headers ได้ แต่ลิงก์ Account/Logout ต้องตั้งค่าการเชื่อมต่อเพิ่มเติม

## รันด้วย Go

ต้องมี Go 1.21 ขึ้นไป ตัวแอปใช้ standard library ไม่มี dependency ภายนอกสำหรับ Go

```sh
go run .
```

## Configuration

ตั้งค่าผ่าน environment ของโปรเซส หรือ `docker run -e NAME=value` แอปไม่ได้โหลดไฟล์ `.env` เอง

| Variable | ค่าเริ่มต้น / การใช้งาน |
| --- | --- |
| `PORT` | `5000` |
| `KEYCLOAK_URL` | URL ของ Keycloak เช่น `https://sso.example.com` ไม่รวม `/realms/...` |
| `REALM` | ชื่อ Keycloak realm สำหรับลิงก์ Account/Logout |
| `CLIENT_ID` | Client ID สำหรับลิงก์ logout |
| `POD_NAME`, `POD_IP` | ข้อมูล Pod ที่ต้องการแสดง |
| `NODE_NAME`, `NODE_IP` | ข้อมูล Node ที่ต้องการแสดง |

ตัวแปรนอกจาก `PORT` จะว่างหากไม่ได้กำหนด ข้อมูล Pod/Node ไม่ได้ถูกค้นหาอัตโนมัติ; บน Kubernetes สามารถส่งค่าผ่าน Downward API ได้

## HTTP behavior

| Request | Response |
| --- | --- |
| `GET /` | หน้า HTML สำหรับตรวจ request และทดลองเรียก API |
| `POST /` | JSON ที่มี `insights`, `headers`, `cookies`, `envs`, ลิงก์ Account/Logout และข้อมูล token หากพบ |
| `OPTIONS /` | `200 OK` พร้อม CORS headers |

ใช้ handler ที่ `/` ครอบคลุม path อื่นด้วย ไม่มีการเก็บประวัติ request และ POST ไม่อ่านหรือสะท้อน request body

```sh
curl -X POST http://localhost:5000/ -H "X-Debug-Example: hello"
```

ส่วน Environment Variables บนหน้าเว็บและฟิลด์ `envs` ยังเป็น map ว่าง โค้ดปัจจุบันอ่านเฉพาะตัวแปรที่ระบุในตาราง configuration

## ตรวจ token และ SSO

```text
Browser -> Ingress / authentication proxy -> OIDC Debugger
                       |
                       +-> Identity provider เช่น Keycloak
```

Proxy ต้องส่ง token มาทาง `X-Auth-Request-Access-Token` หรือ `Authorization` แอปตัด prefix `Bearer ` หากมี แล้วถอด JWT payload เพื่อแสดงผล ควรส่งเพียง header เดียว เพราะหากมีทั้งสอง header โค้ดไม่ได้กำหนดลำดับความสำคัญ

การถอด payload ไม่ตรวจ signature, issuer, audience หรือ expiration และไม่ใช่หลักฐานว่า token ใช้งานได้ การตรวจสิทธิ์ต้องทำโดย proxy หรือบริการที่รับผิดชอบ

ปุ่ม Logout ชี้ไป `/oauth2/sign_out` บน host เดียวกัน จากนั้นส่งต่อไปยัง Keycloak logout endpoint จึงต้องมี OAuth2 Proxy route และตั้งค่า redirect ที่เกี่ยวข้องให้ตรงกัน ปุ่ม Manage Account ชี้ไปหน้า account ของ realm

## API tester และ CORS

หน้าเว็บมีปุ่ม GET, POST และ OPTIONS โดย URL เริ่มต้นเติม `api.` หน้า host ปัจจุบัน แก้ URL ให้ตรงกับ API ที่ต้องการทดสอบได้

Tester ส่ง token ผ่าน header `app-token` และใช้ `Content-Type: application/json` สำหรับ GET/POST จะมีการเรียก OPTIONS แยกก่อน แต่การเรียกนี้ไม่ใช่การแสดง automatic CORS preflight ของ browser ให้ใช้ Network panel ของ browser ตรวจ preflight จริง

ฝั่ง server ตอบ `Access-Control-Allow-Origin: *` พร้อม methods `GET, POST, OPTIONS` และ headers `app-token, Content-Type, Authorization` ทั้งนี้การอนุญาต `app-token` ผ่าน CORS ไม่ได้ทำให้แอปใช้ header นี้ถอด JWT; ตัว decoder อ่านเฉพาะสอง headers ที่ระบุข้างต้น

## ข้อจำกัดในการใช้งาน

- ไม่มี authentication ในตัว และแสดง headers, cookies และ token ที่ได้รับ ควรจำกัดการเข้าถึงและใช้ token สำหรับทดสอบ
- Headers ที่แสดงไม่ใช่ข้อมูลยืนยันตัวตนที่เชื่อถือได้ หากไม่มี proxy ที่ตรวจสอบและควบคุม headers เหล่านั้น
- หน้าเว็บโหลด Prism CSS/JavaScript จาก CDN จึงต้องเข้าถึง CDN ได้เพื่อใช้ syntax highlighting และส่วน UI ที่เรียก Prism
- ไม่มี token validation, OIDC discovery, token exchange หรือ persistent storage

## โครงสร้างและคำสั่งพัฒนา

| ไฟล์ | หน้าที่ |
| --- | --- |
| `main.go` | HTTP handler, HTML/JavaScript template และ JWT payload decoder |
| `go.mod` | Go module |
| `Dockerfile` | Build static Go binary แล้วรันบน Alpine |

```sh
go build -o oidc-debugger .
go vet ./...
```

ปัจจุบันยังไม่มี automated tests ก่อนส่งการแก้ไขให้ตรวจทั้งหน้า HTML และ POST JSON ด้วยข้อมูลทดสอบ และอย่า commit token หรือ cookies จริง
