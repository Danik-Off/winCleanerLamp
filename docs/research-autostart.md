# Исследование: управление автозагрузкой Windows

Сводка по открытым источникам о том, где Windows хранит список программ,
запускаемых при входе в систему, и как штатно (без стороннего API) включать
и выключать отдельные записи так же, как это делает вкладка «Автозагрузка»
диспетчера задач.

## Источники автозапуска (где искать)

Три группы источников, все проверяются диспетчером задач Windows:

1. **Реестр — Run/RunOnce** (4 ключа):
   - `HKCU\Software\Microsoft\Windows\CurrentVersion\Run` — для текущего пользователя, при каждом входе.
   - `HKCU\Software\Microsoft\Windows\CurrentVersion\RunOnce` — один раз, значение само удаляется после выполнения.
   - `HKLM\Software\Microsoft\Windows\CurrentVersion\Run` — для всех пользователей.
   - `HKLM\Software\Microsoft\Windows\CurrentVersion\RunOnce` — для всех пользователей, один раз.
   - Плюс 32-битные зеркала под `HKLM\Software\WOW6432Node\Microsoft\Windows\CurrentVersion\{Run,RunOnce}` — 32-битные приложения на 64-битной Windows пишут сюда, а не в основной `HKLM\...\Run`.
2. **Папки автозагрузки**:
   - Пользовательская: `%APPDATA%\Microsoft\Windows\Start Menu\Programs\Startup`.
   - Общая (все пользователи): `%ProgramData%\Microsoft\Windows\Start Menu\Programs\Startup`.
   - Ярлыки/exe в этих папках запускаются при входе автоматически — Explorer сам их выполняет.
3. **Планировщик заданий (Task Scheduler)** с триггером «При входе в систему» (`LogonTrigger`). Диспетчер задач НЕ показывает их во вкладке «Автозагрузка» в старых версиях, но многие производители (принтеры, антивирусы, апдейтеры типа Adobe/NVIDIA/Google Update) используют именно задания планировщика, а не Run-ключи — по факту это тоже автозапуск и тоже нужно уметь искать.

**Сознательно не включено** (слишком рискованно/не является типичным
«автозапуском приложений», трогать нельзя без экспертных знаний):
служба Windows (`services.msc`) со стартом «Автоматически» — отключение
системной службы может сломать ОС; `Winlogon\Userinit`/`Shell` — критичны для
самого входа в систему; `Session Manager\BootExecute` — уровень ядра;
Active Setup (`HKLM\SOFTWARE\Microsoft\Active Setup\Installed Components`) —
редко используется вне корпоративной среды и легко сломать профиль.

## Как Windows хранит состояние «включено/выключено»

Ключевой факт (подтверждён несколькими независимыми источниками, включая
блог Windows Incident Response с анализом бинарного формата): **выключение
записи через диспетчер задач НЕ удаляет её** из `Run`/папки автозагрузки —
Windows лишь запоминает решение пользователя в отдельном месте:

- `HKCU\Software\Microsoft\Windows\CurrentVersion\Explorer\StartupApproved\Run` —
  состояние для записей из `HKCU\...\Run` и `HKLM\...\Run` (64-битные).
- `HKCU\...\StartupApproved\Run32` — состояние для записей из
  `HKLM\...\WOW6432Node\...\Run` (32-битные).
- `HKCU\...\StartupApproved\StartupFolder` — состояние для файлов в обеих
  папках автозагрузки (пользовательской и общей), ключ — имя файла.

Значение — **REG_BINARY** длиной 12 байт:

- Включено: `02 00 00 00 00 00 00 00 00 00 00 00`.
- Выключено: `03 00 00 00 XX XX XX XX XX XX XX XX` (первые 4 байта после `03`
  всегда нули, последние 8 байт — таймстамп момента выключения в некоторых
  версиях Windows; для функциональности сам факт `03` в первом байте
  достаточен — большинство инструментов, включая наш, пишут нули и в эти
  байты).
- Если значения для имени вообще нет в `StartupApproved\...` — запись
  считается включённой (это поведение самого Explorer/Task Manager).

Это значит: **включение/выключение делается одной операцией записи в
реестр** (`reg add ... /t REG_BINARY /d ...`), без удаления самой команды
автозапуска — обратимо в один клик, ничего не теряется.

Для заданий планировщика своего аналога `StartupApproved` нет — там состояние
хранится в самом задании (`State: Ready | Disabled`), включается/выключается
через `schtasks /Change /TN "<путь>" /Enable|/Disable`.

## План реализации (соответствует уже принятой архитектуре проекта)

Проект принципиально не использует сторонние Go-модули (`README.md`) — вся
работа с реестром/планировщиком идёт через `reg.exe`/`schtasks.exe`/
`powershell.exe`, как уже сделано для остальных функций (`orphan.go`,
`leftovers.go`). Автозагрузка следует тому же паттерну:

- **Листинг** реестровых Run-ключей — `reg query "<ключ>"` (как в
  `installedProgramNames()`), полей достаточно, чтобы получить имя значения
  и команду.
- **Листинг заданий планировщика с триггером входа** — через PowerShell
  (`Get-ScheduledTask` + фильтр по `CimClassName -eq 'MSFT_TaskLogonTrigger'`,
  вывод `ConvertTo-Json`), а не текстовый `schtasks /query /v /fo csv» —
  потому что вывод `schtasks` **локализован** (на русской Windows «At logon»
  превращается в русскую строку), а имена классов CIM в PowerShell — нет.
  Учитывая, что интерфейс проекта на русском и рассчитан на русские
  локализации Windows, этот момент важен и мог бы незаметно сломать фичу
  только на англоязычных сборках при обратном подходе.
- **Чтение/запись `StartupApproved`** — `reg query`/`reg add` с
  `/t REG_BINARY`, как описано выше.
- **GUI-только**: команды регистрируются в CLI (`--autostart-list`,
  `--autostart-set`, всегда с `--json`), но не документируются подробно в
  `usage()`/`docs/cli.md` как «полноценная CLI-фича» — по требованию
  пользователя это функция для GUI (тем, кому нужен именно CLI-доступ к
  автозагрузке, проще и привычнее открыть `regedit`/`taskschd.msc` напрямую).
  Технически это по-прежнему тот же бинарник (иначе GUI не сможет его
  вызвать — архитектура «GUI = обёртка над одним exe» не меняется), но без
  «витринного» CLI UX.

## Источники

- [Run and RunOnce Registry Keys — Microsoft Learn](https://learn.microsoft.com/en-us/windows/win32/setupapi/run-and-runonce-registry-keys)
- [Windows Incident Response: StartupApproved\Run, pt II](http://windowsir.blogspot.com/2022/07/startupapprovedrun-pt-ii.html)
- [Windows OS Optimization Essentials, Part 4: Startup Items — Nutanix](https://www.nutanix.com/en_sg/blog/windows-os-optimization-essentials-part-4-startup-items)
- [Registry Run Keys / Startup Folder — Red Team Notes / MITRE ATT&CK T1547.001](https://attack.mitre.org/techniques/T1547/001/)
- [Windows Automatic Startup Locations — gHacks](https://www.ghacks.net/2016/06/04/windows-automatic-startup-locations/)
- [reg add command reference — Microsoft Learn](https://learn.microsoft.com/en-us/windows-server/administration/windows-commands/reg-add)
- [How to Enable and Disable Scheduled Tasks on Windows](https://www.enigmasoftware.com/how-to-enable-and-disable-scheduled-tasks-on-windows/)
- [Wow6432Node registry key — Microsoft Learn](https://learn.microsoft.com/en-us/troubleshoot/windows-client/application-management/wow6432node-registry-key-present-32-bit-machine)
