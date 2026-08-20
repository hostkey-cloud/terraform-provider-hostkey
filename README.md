# Hostkey | Terraform Provider

[![Terraform Registry](https://img.shields.io/badge/registry-hostkey--cloud%2Fhostkey-623CE4)](https://registry.terraform.io/providers/hostkey-cloud/hostkey/latest)

Terraform-провайдер для [Hostkey](https://hostkey.ru/): VPS, dedicated, GPU и DNS через [InvAPI](https://hostkey.ru/documentation/apidocs/api_index/).

English: [README.en.md](README.en.md) · Registry: [`hostkey-cloud/hostkey`](https://registry.terraform.io/providers/hostkey-cloud/hostkey/latest)

## Документация

Полное описание атрибутов — в [`docs/`](docs/) (страницы [Terraform Registry](https://registry.terraform.io/providers/hostkey-cloud/hostkey/latest/docs)). Примеры: [`examples/`](examples/).

### Ресурсы

| Ресурс | Назначение |
|--------|------------|
| [`hostkey_server`](docs/resources/server.md) | Заказ и управление сервером (VPS, dedic, GPU, vGPU) |
| [`hostkey_server_ip`](docs/resources/server_ip.md) | Дополнительный IPv4 на сервере |
| [`hostkey_ssh_key`](docs/resources/ssh_key.md) | SSH-ключ в хранилище аккаунта InvAPI |
| [`hostkey_dns_domain`](docs/resources/dns_domain.md) | DNS-зона (PowerDNS) |
| [`hostkey_dns_record`](docs/resources/dns_record.md) | Запись в DNS-зоне |

### Источники данных

| Источник | Назначение |
|----------|------------|
| [`hostkey_presets`](docs/data-sources/presets.md) | Список пресетов (`presets/list`) |
| [`hostkey_preset`](docs/data-sources/preset.md) | Один пресет по id или имени |
| [`hostkey_oses`](docs/data-sources/oses.md) | ОС для пресета или сервера |
| [`hostkey_traffic_plans`](docs/data-sources/traffic_plans.md) | Планы трафика для локации / пресета |
| [`hostkey_software`](docs/data-sources/software.md) | ПО маркетплейса для пресета |
| [`hostkey_ssh_keys`](docs/data-sources/ssh_keys.md) | SSH-ключи аккаунта |
| [`hostkey_dns_domains`](docs/data-sources/dns_domains.md) | DNS-зоны аккаунта |

Один ресурс `hostkey_server` покрывает весь каталог: `vm.*`, `vds.*`, `bm.*`, `gpu.*`, `vgpu.*`. Dedicated / GPU — [docs/resources/server.md](docs/resources/server.md).

## Требования

* [Terraform](https://developer.hashicorp.com/terraform/tutorials/aws-get-started/install-cli) **>= 1.0**
* API-ключ **аккаунта** InvAPI (тип `Any`): [документация](https://hostkey.ru/documentation/account/api_key_account/)
* Заказ сервера — **платный**; deploy может занять до ~90 минут

## Быстрый старт

### 1. Конфигурация

Создайте каталог проекта и файл `main.tf`:

```hcl
terraform {
  required_providers {
    hostkey = {
      source  = "hostkey-cloud/hostkey"
      version = "~> 0.1"
    }
  }
  required_version = ">= 1.0"
}

provider "hostkey" {
  region = var.hostkey_region
  # api_key — из HOSTKEY_API_KEY (см. ниже) или явно: api_key = var.hostkey_api_key
}

variable "hostkey_region" {
  type        = string
  description = "InvAPI endpoint: RU (.ru) или COM (.com). Не путать с location_name (ДЦ)."
  default     = "RU"
}

variable "root_pass" {
  type        = string
  sensitive   = true
  description = "Root-пароль (8–30 символов; см. docs/resources/server.md)."
}

# Сверьте имена в каталоге перед заказом (read-only, бесплатно):
data "hostkey_presets" "pico" {
  location = "NL"
  name     = "vm.pico"
}

data "hostkey_traffic_plans" "vm" {
  location    = "NL"
  instance_id = data.hostkey_presets.pico.presets[0].id
}

resource "hostkey_server" "web" {
  preset_name       = "vm.pico"
  location_name     = "NL"
  os_name           = "Ubuntu 22.04"
  traffic_plan_name = "3 TB / 1 Gbps VM"
  deploy_period     = "monthly"
  root_pass         = var.root_pass
  power_state       = "on"

  # Для destroy: 0 — в конце периода, 1 — немедленно (если разрешено аккаунтом)
  cancellation_type   = 1
  cancellation_reason = "terraform"

  timeouts {
    create = "90m"
    delete = "30m"
  }
}

output "server_id" {
  value = hostkey_server.web.id
}

output "main_ipv4" {
  value = hostkey_server.web.main_ipv4
}
```

Скопируйте [`examples/basic/terraform.tfvars.example`](examples/basic/terraform.tfvars.example) → `terraform.tfvars` (файл в `.gitignore`, **не коммитить**):

```hcl
root_pass = "StrongPass1%"
```

### 2. API-ключ

InvAPI → **Имя пользователя → API ключи**: [hostkey.ru](https://hostkey.ru/documentation/account/api_key_account/) · [invapi.hostkey.ru](https://invapi.hostkey.ru).

**Способ 1 — переменная окружения (рекомендуется):**

```powershell
$env:HOSTKEY_API_KEY = "ваш-ключ"
```

```bash
export HOSTKEY_API_KEY="your-key"
```

**Способ 2 — в провайдере** (добавьте variable и `terraform.tfvars`):

```hcl
variable "hostkey_api_key" {
  type      = string
  sensitive = true
}

provider "hostkey" {
  region  = var.hostkey_region
  api_key = var.hostkey_api_key
}
```

Алиасы env: `HOSTKEY_API_TOKEN`. Переопределение URL: `HOSTKEY_BASE_URL` / `HOSTKEY_API_URL`.

### 3. Если `registry.terraform.io` недоступен (RU)

HashiCorp блокирует часть сетей. Провайдер тот же: `source = "hostkey-cloud/hostkey"`. Аккаунт в Yandex Cloud **не нужен**.

Создайте файл CLI Terraform:

- Linux / macOS: `~/.terraformrc`
- Windows: `%APPDATA%\terraform.rc`

```hcl
provider_installation {
  network_mirror {
    url     = "https://terraform-mirror.yandexcloud.net/"
    include = ["registry.terraform.io/*/*"]
  }
  direct {
    exclude = ["registry.terraform.io/*/*"]
  }
}
```

Затем `terraform init -upgrade`. Если рядом стоит `dev_overrides` на локальный бинарник — для проверки Registry его уберите.

### 4. Init, validate, plan, apply

```bash
terraform init
terraform validate   # Success! The configuration is valid.
terraform plan
terraform apply
```

Terraform покажет план изменений и запросит подтверждение — введите **`yes`** и Enter.

Заказ **платный**. Создание сервера — асинхронное (обычно десятки минут, timeout по умолчанию 90m).

### 5. Destroy

```bash
terraform destroy
```

Снова подтвердите **`yes`**. Вызывается `whmcs/request_cancellation` с `cancellation_type` / `cancellation_reason` из ресурса.

## Особенности Hostkey (InvAPI)

* **`region`** (провайдер) — endpoint API (`invapi.hostkey.ru` / `.com`), default в схеме — `COM`. **`location_name`** (ресурс) — дата-центр (`NL`, `US`, `RU`, …).
* Имена **`preset_name` / `os_name` / `traffic_plan_name`** — **точно как в InvAPI**, не как короткие подписи в панели (`bm.v2-promo`, не `v2-promo`).
* Перед заказом: `data.hostkey_presets` + `data.hostkey_traffic_plans` с **`instance_id`** = id пресета.
* У dedicated часто **два плана с одним `name` и разной `price`** — используйте подсказку из панели (`- FREE`, `(10000 P)`) или `traffic_plan_id`.
* **`disk_mirror`** — только если в `presets/list` у пресета **2+ диска**; на однодисковых (в т.ч. `bm.v2-promo`) поле **не задавать**.
* Поля заказа только типизированные атрибуты (`extra_order_params` удалён).
* BM / GPU / vGPU, RAID, IPv6, reinstall — [docs/resources/server.md](docs/resources/server.md).

Локальная сборка без Registry: `go install` + [dev_overrides](examples/dev-terraform.rc) — [CONTRIBUTING.md](CONTRIBUTING.md).

## Import

```bash
terraform import hostkey_server.web 12345
```

Import по числовому id InvAPI — подробнее в [Registry: hostkey_server → Import](https://registry.terraform.io/providers/hostkey-cloud/hostkey/latest/docs/resources/server#import).

## Устранение неполадок

См. [Registry: Troubleshooting](https://registry.terraform.io/providers/hostkey-cloud/hostkey/latest/docs#troubleshooting).

## Разработка

* [CONTRIBUTING.md](CONTRIBUTING.md)
* [SECURITY.md](SECURITY.md)
* [CHANGELOG.md](CHANGELOG.md)
* Лицензия [MPL-2.0](LICENSE)
