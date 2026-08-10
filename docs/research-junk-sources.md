# Исследование: где на Windows накапливается мусор

Сводка по открытым источникам (Microsoft Learn, форумы поддержки, документация
BleachBit/winapp2.ini, обсуждения на Microsoft Q&A и профильных форумах) о
местах на диске, где Windows и приложения оставляют временные/кеш-файлы.
Список используется как источник для новых категорий в `internal/cleaner/targets.go`.
Ссылки — в конце файла.

## Методология классификации

Каждое место отнесено к одной из трёх групп:

- **Безопасно (авто)** — можно чистить без специального предупреждения; данные
  пересоздаются приложением/ОС. Добавлено как обычная безопасная категория.
- **Агрессивно** — можно чистить, но с оговорками (блокировки файлов,
  повторная загрузка данных занимает время/трафик, потенциальная потеря
  недавнего контекста). Добавлено как категория с `Aggressive: true`.
- **Не трогать (только информация)** — официально не рекомендуется удалять
  (Microsoft прямо предупреждает), или удаление ломает функциональность
  (ремонт/удаление программ, история для служб). Не добавляется как
  удаляемая категория — только объясняется пользователю (`--sysinfo`-style).

## Уже покрыто в проекте (для справки)

`%TEMP%`, `C:\Windows\Temp`, Prefetch, SoftwareDistribution\Download, CBS-логи,
WER, crash dumps, thumbnail/icon cache, INetCache, Recent/Jump Lists, кеши
браузеров (Chrome/Edge/Brave/Firefox), кеши мессенджеров (Teams/Discord/
Slack/Telegram/Skype), Spotify, VS Code, JetBrains, Office File Cache, Adobe
Media Cache, кеши пакетных менеджеров (NuGet/pip/npm/Yarn/go-build/Gradle),
GPU shader cache (NVIDIA/AMD/DX), корзина, DNS-кеш, Windows.old,
$WINDOWS.~BT/~WS, event-логи, старые Downloads, Maven.

## Новые источники по итогам исследования

### Безопасно (добавлено как обычные категории)

| Путь | Что это | Источник |
|---|---|---|
| `C:\Windows\CSC` | Client-Side Caching — кеш автономных файлов (Offline Files). На домашних ПК почти всегда пуст, но на корпоративных/доменных машинах может расти. | общие Windows-гайды по CSC |
| `%LOCALAPPDATA%\Microsoft\Terminal Server Client\Cache` | Кеш миниатюр удалённого рабочего стола (RDP bitmap cache) | стандартное расположение RDP-клиента |
| `%LOCALAPPDATA%\Microsoft\Windows\Notifications` | База данных уведомлений действий (wpndatabase.db и логи) | известное расположение, пересоздаётся |
| `%APPDATA%\Composer\cache` | Кеш пакетного менеджера Composer (PHP) | аналогично npm/pip — восстанавливается |
| `%LOCALAPPDATA%\pnpm-store`, `%LOCALAPPDATA%\pnpm\store` | Локальное хранилище пакетов pnpm | аналог npm-кеша |
| `%APPDATA%\Zoom\logs`, `%APPDATA%\Zoom\data\Cache` | Логи и кеш Zoom Desktop | winapp2.ini-класс записей для чат/конференц-клиентов |
| `%APPDATA%\Postman\Cache` | Кеш Postman | аналогично |
| `%APPDATA%\Signal\Cache`, `%APPDATA%\Signal\logs` | Кеш и логи Signal Desktop | аналогично Telegram/Discord |
| `%APPDATA%\obs-studio\logs`, `crash-reports` | Логи и краш-репорты OBS Studio | известное расположение |
| `%LOCALAPPDATA%\Docker\log` | Логи Docker Desktop (не сам образ/VHDX — тот трогать нельзя) | Docker Desktop docs |
| `%ProgramData%\Microsoft\Network\Downloader` | Кеш заданий BITS/Delivery Optimization (доп. к уже покрытому DeliveryOptimization\Cache) | Windows BITS docs |

### Агрессивно (добавлено с `Aggressive: true`, т.к. дороже восстановить или чаще блокируется)

| Путь | Что это | Почему агрессивно |
|---|---|---|
| `%LOCALAPPDATA%\Microsoft\Windows\WebCache\*.dat` (WebCacheV01.dat) | ESE-база кеша WinINet/IE/поиска | Часто заблокирована активными процессами (Search, Host Process for Windows Tasks); в редких случаях требует безопасного режима. Пересоздаётся, но риск ошибок блокировки выше обычного. |
| `%USERPROFILE%\go\pkg\mod` | Кеш загруженных Go-модулей (`go mod download`) | Десятки ГБ у активных Go-разработчиков; повторная загрузка требует сети/времени. Существующий `go-build-cache` — это ДРУГОЙ кеш (build cache), эта запись — кеш модулей. |
| `%USERPROFILE%\.cargo\registry\cache`, `%USERPROFILE%\.cargo\registry\src` | Кеш зарегистрированных crate-пакетов Rust/Cargo (не трогает `.cargo\bin`) | Аналогично go pkg mod — большой объём, требует сети для восстановления. |
| `%USERPROFILE%\.cache\huggingface` | Кеш загруженных моделей Hugging Face (ML) | Модели могут быть по несколько ГБ каждая; повторная загрузка — трафик и время. |

### Не трогать (только информация, не добавляется как категория)

| Путь | Почему нельзя |
|---|---|
| `C:\ProgramData\Package Cache` | Microsoft официально предупреждает: используется для ремонта/удаления/изменения Visual Studio и других MSI-based программ офлайн. Официальная рекомендация — **не удалять**, а отключить кеширование (`--nocache` при установке) или перенести папку. |
| `C:\Windows\Installer` (уже упомянуто в sysinfo) | Кеш MSI-установщиков — поломает repair/uninstall программ. |
| `C:\Windows\WinSxS` | Хранилище компонентов — только `dism /Cleanup-Image`, не прямое удаление. |
| Unreal Engine `DerivedDataCache` | Путь специфичен для каждого проекта/версии движка (нет единого фиксированного расположения), в UE5.4+ вообще может быть в Zen Store — не универсализируется под одну Target-запись; пользователю стоит чистить вручную через Editor. |
| Docker Desktop образ (WSL2 VHDX) | Не файловый кеш, а виртуальный диск — удаление вручную требует `wsl --unregister`/настроек Docker Desktop, не «удаление файла». |

## Автозагрузка — отдельное исследование

См. [`docs/research-autostart.md`](./research-autostart.md).

## Источники

- [Run and RunOnce Registry Keys — Microsoft Learn](https://learn.microsoft.com/en-us/windows/win32/setupapi/run-and-runonce-registry-keys)
- [Disable or move the package cache — Visual Studio docs](https://learn.microsoft.com/en-us/visualstudio/install/disable-or-move-the-package-cache?view=vs-2022)
- [What is C:\ProgramData\Package Cache? — Microsoft Q&A](https://learn.microsoft.com/en-us/answers/questions/4352991/what-is-c-programdatapackage-cache-can-it-be-delet)
- [Is it safe to delete SoftwareDistribution / Package Cache — Microsoft Q&A](https://learn.microsoft.com/en-us/answers/questions/3790161/is-it-safe-to-delete-files-in-c-programdatapackage)
- [WebCacheV01.dat discussion — Microsoft Q&A](https://learn.microsoft.com/en-us/answers/questions/4335865/system32-webcache-webcachev01-dat)
- [Winapp2.ini — MoscaDotTo/Winapp2 (GitHub)](https://github.com/MoscaDotTo/Winapp2)
- [Winapp2.ini Guide — BleachBit docs](https://docs.bleachbit.org/doc/winapp2ini.html)
- [Using Derived Data Cache in Unreal Engine — Epic Developer Community](https://dev.epicgames.com/documentation/en-us/unreal-engine/using-derived-data-cache-in-unreal-engine)
- [How to Free Up Disk Space as a Developer — DEV Community](https://dev.to/riponcm/how-to-free-up-disk-space-as-a-developer-clear-hugging-face-npm-conda-and-docker-caches-3h21)
- [Windows SoftwareDistribution folder guide](https://www.solvemix.com/computer/windows-11/windows-softwaredistribution-folder-guide-and-purpose.html)
