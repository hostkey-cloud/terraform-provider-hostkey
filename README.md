# Hostkey | Terraform Provider

Terraform позволяет управлять инфраструктурой [Hostkey](https://hostkey.com/) (VPS и dedicated) через [InvAPI](https://hostkey.com/documentation/apidocs/api_index/) с помощью конфигурации HCL и планов изменений.

English version: [README.en.md](README.en.md).

Подробнее о Terraform: [developer.hashicorp.com/terraform](https://developer.hashicorp.com/terraform/docs).

## Документация

Параметры ресурсов и источников данных — в каталоге [`docs/`](docs/) (страницы для [Terraform Registry](https://registry.terraform.io/)).

### Ресурсы

* [hostkey_server](docs/resources/server.md) — заказ и управление сервером
* [hostkey_server_ip](docs/resources/server_ip.md) — дополнительный IPv4
* [hostkey_ssh_key](docs/resources/ssh_key.md) — SSH-ключ в хранилище аккаунта
* [hostkey_dns_domain](docs/resources/dns_domain.md) — DNS-зона
* [hostkey_dns_record](docs/resources/dns_record.md) — DNS-запись

### Источники данных

* [hostkey_presets](docs/data-sources/presets.md) — список пресетов
* [hostkey_preset](docs/data-sources/preset.md) — один пресет по id
* [hostkey_oses](docs/data-sources/oses.md) — операционные системы
* [hostkey_traffic_plans](docs/data-sources/traffic_plans.md) — тарифы трафика
* [hostkey_software](docs/data-sources/software.md) — ПО маркетплейса
* [hostkey_ssh_keys](docs/data-sources/ssh_keys.md) — SSH-ключи аккаунта
* [hostkey_dns_domains](docs/data-sources/dns_domains.md) — DNS-домены

Примеры конфигураций: [`examples/`](examples/).

## Быстрый старт

### 1. Установите Terraform

По [официальной инструкции](https://developer.hashicorp.com/terraform/tutorials/aws-get-started/install-cli). Нужна версия **>= 1.0**.

### 2. Создайте конфигурацию

Каталог, например `hostkey-terraform`, файл `main.tf`:

```hcl
terraform {
  required_providers {
    hostkey = {
      source  = "hkadm/hostkey"
      version = "~> 0.1"
    }
  }
  required_version = ">= 1.0"
}

provider "hostkey" {
  region = "RU" # или COM — биллинг/API (.ru / .com), не дата-центр
}

resource "hostkey_server" "web" {
  preset_name       = "vm.pico"
  location_name     = "NL"
  os_name           = "Ubuntu 22.04"
  traffic_plan_name = "3 TB / 1 Gbps VM"
  deploy_period     = "monthly"
  root_pass         = var.root_pass
}
```

> Пока провайдер не опубликован в Registry, для разработки используйте `go install` и [dev_overrides](examples/dev-terraform.rc) — см. [CONTRIBUTING.md](CONTRIBUTING.md).

### 3. API-ключ

Создайте ключ в InvAPI: **Configuration → API keys**  
([документация Hostkey](https://hostkey.com/documentation/controlpanel/apikey/)).

```powershell
$env:HOSTKEY_API_KEY = "ваш-ключ"
```

```bash
export HOSTKEY_API_KEY="your-key"
```

Либо в конфигурации:

```hcl
provider "hostkey" {
  region  = "RU"
  api_key = var.hostkey_api_key
}
```

Также принимаются `HOSTKEY_API_TOKEN` (алиас ключа) и `HOSTKEY_BASE_URL` / `HOSTKEY_API_URL` (базовый URL InvAPI).

**Важно:** `region` выбирает endpoint InvAPI (`.ru` / `.com`). Дата-центр задаётся в ресурсе как `location_name` (например `NL`, `RU`).

### 4. Plan / Apply / Destroy

```bash
terraform init
terraform plan
terraform apply
terraform destroy
```

Заказ сервера **платный**. Deploy может занять от десятков минут до полутора часов.  
`destroy` вызывает отмену услуги (`whmcs/request_cancellation`). Тип отмены можно задать через `cancellation_type` / `cancellation_reason` у `hostkey_server`.

## Аутентификация

| Способ | Описание |
|--------|----------|
| `HOSTKEY_API_KEY` или `HOSTKEY_API_TOKEN` | Рекомендуется |
| `provider.api_key` | Явно в HCL (лучше через переменную, не в git) |

Провайдер обменивает ключ на сессионный токен (`auth/login`) и использует его в запросах к InvAPI.

## Разработка и релиз

* Участие в разработке: [CONTRIBUTING.md](CONTRIBUTING.md)
* Публикация релиза: [RELEASE.md](RELEASE.md)
* Безопасность: [SECURITY.md](SECURITY.md)
* История изменений: [CHANGELOG.md](CHANGELOG.md)
* Лицензия: [MPL-2.0](LICENSE)

## License

MPL-2.0
