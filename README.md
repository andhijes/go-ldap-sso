# Go SSO & LDAP Login Demo

This project is a Go-based authentication demo that supports two login methods:

- ✅ Login via **SSO (SAML)** — redirects users to [MockSAML.com](https://mocksaml.com)
- ✅ Login via **LDAP** — using an embedded OpenLDAP server via Docker

The purpose of this project is to demonstrate how to integrate **SSO** and **LDAP** login mechanisms in a Go application.

---

## 🚀 Getting Started

### 1. Copy Environment Variables

```bash
cp .env.example .env
````

Update the `.env` file as needed, including the `DB_NAME` for your local database.

---

### 2. Start LDAP Server

```bash
./setup-up.sh
```

This script will:

* Start the LDAP container
* Seed it with predefined users from `ldif/` files

---

### 3. Prepare Go Modules

Install and tidy Go dependencies:

```bash
go mod tidy
```

---

### 4. Initialize Database

Make sure your database server (e.g., PostgreSQL or MySQL) is running, then manually create a new database with the name specified in your `.env` file under `DB_NAME`.

Example for PostgreSQL:

```bash
createdb your_db_name
```

Or MySQL:

```bash
mysql -u root -p -e "CREATE DATABASE your_db_name;"
```

---

### 5. Run Database Migrations

```bash
go run cmd/main.go migrate up
```

This will create necessary tables and schema in your database.

---

### 6. Run Database Seeder

```bash
go run cmd/main.go seed run
```

This will populate your database with initial data (e.g., test users or roles).

---

### 7. Run the Go Application

```bash
go run cmd/main.go api
```

The app will be available at:
👉 `http://localhost:8080`

---

## 🧪 Try the Demo

### LDAP Admin UI

Visit:
👉 `http://localhost:8081`

Login:

* **Username**: `admin`
* **Password**: `admin1234`

### Web App Login (localhost:8080)

You will see two options:

* 🔹 **Login with SSO**

  * Redirects to [mocksaml.com](https://mocksaml.com)
* 🔹 **Login with LDAP**

  * Use:

    * **Username**: `admin`
    * **Password**: `admin1234`

---

### 8. Stop and Clean Up

To shut everything down and remove volumes:

```bash
./setup-down.sh
```

---

## 📁 Project Structure

```
.
├── .env.example              # Example environment variables
├── setup-up.sh               # Start LDAP + load LDIFs
├── setup-down.sh             # Stop LDAP container
├── docker-compose.osixia.yml # LDAP container definition
├── ldif/                     # LDAP seed files
├── cmd/main.go               # Go entrypoint (API, migrations, seeder)
└── internal/...              # Application source code
```

---

## 🧪 Demo Accounts

| Login Type | Username | Password                 |
| ---------- | -------- | ------------------------ |
| LDAP       | admin    | admin1234                |
| SSO        | -        | Redirect to mocksaml.com |

---

## 🪪 License

This project is provided for learning and development purposes only.

