# glowbyte-hw

- [Задания](#задания)
- [Установка Postgres](#установка-postgres)

## Задания

1. Установить где-то у себя Постгрес (на виртуальной машине, например)
2. На пункте 1 научиться делать бекап и рестор. Уметь отвечать на вопросы, а что там происходит.
3. Научиться отвечать на вопросы, что такое вакуум, вакуум фулл, аналайз и что будет, если их не делать?
4. Типы блокировок - какие есть, для чего нужны, где посмотреть?
5. Что такое MVCC, для чего он нужен, и как реализован в Постгресе?
6. Изучить сервисные таблицы (pg_stat_activity, например). Что там лежит? В каких случаях туда надо идти смотреть?

## Установка Postgres

Конфигурация сервисов `postgres` и `pgadmin` описана в `docker-compose.yml`.
Так же есть сервис приложения на Golang для заполнения бд тестовыми данными.

### Запуск

Только `postgres` и `pgadmin`:

```shell
docker compose up
```

С заполнение базы данных:

```shell
docker compose --profile seed up --build
```

После запуска `pgadmin` будет доступен на [localhost:5050](http://localhost:5050)

### Шаблон `.env`

```env
POSTGRES_PASSWORD=
POSTGRES_DB=
PGADMIN_DEFAULT_EMAIL=
PGADMIN_DEFAULT_PASSWORD=
```

Переименуйте и заполните файл `.env.template`.

## Backup & Restore

Документация: [postgresql.org/docs/backup](https://www.postgresql.org/docs/current/backup.html)

### SQL dump

> The idea behind this dump method is to generate a file with SQL commands that, when fed back to the server, will recreate the database in the same state as it was at the time of the dump.

#### Особенности

- Вывод в `stdout`.
- Позволяет создавать резервные копии с любого удаленного хоста, имеющего доступ к базе данных.
- Требует доступ на чтение ко всем таблицам. Полный бекап базы данных обычно требует привилегий супер-пользователя. Частичные бекапы можно делать например с помощью `-n schema` или `-t table`.
- Вывод, в отличие от других методов, совместим с более новыми версиями PostgreSQL.
- Единственный метод, подходящий для переноса баз данных между различными архитектурами (например 32-бит -> 64-бит).
- Внутренняя консистентность: представляет собой снимок базы данных в начале процесса создания дампа.
- Не блокирует большинство операций с базой данных.

#### Backup

```shell
# docker exec -t <container> pg_dumpall -c -U <role> > <filename>.sql

docker exec -t glowbyte-pg pg_dumpall -c -U postgres > dump_`date +%Y-%m-%d"_"%H_%M_%S`.sql
```

#### Restore

```shell
# cat <filename>.sql | docker exec -i <container> psql -U <role>

cat $(ls dump*.sql -rt created | head -n1) | docker exec -i glowbyte-pg psql -U postgres
```

### File System Level Backup

> The backup strategy is to directly copy the files that PostgreSQL uses to store the data in the database. You can use whatever method you prefer for doing file system backups.

#### Особенности

- Сервер базы данных должен быть выключен и при создании бекапа, и при восстановлении.
- Работает только для полного резервного копирования и восстановления всего кластера.
- Размер файла обычно больше, чем у SQL dump, но процесс может быть быстрее.

#### Backup

Пример из [документации](https://www.postgresql.org/docs/current/backup-file.html):

```shell
tar -cf backup.tar /usr/local/pgsql/data
```

Поскольку в этом проекте используется Docker, volume `/var/lib/postgresql/data` и так прокинут на хост в директорию `./docker/pgdata`. Так что можно работать с ним:

```shell
tar -cf backup.tar ./docker/pgdata
```

#### Restore

```shell
tar -xzf backup.tar -C .
```

### Continuous Archiving and Point-in-Time Recovery (PITR)

> At all times, PostgreSQL maintains a write ahead log (WAL) in the pg_wal/ subdirectory of the cluster's data directory. The log records every change made to the database's data files. This log exists primarily for crash-safety purposes: if the system crashes, the database can be restored to consistency by “replaying” the log entries made since the last checkpoint. However, the existence of the log makes it possible to use a third strategy for backing up databases: we can combine a file-system-level backup with backup of the WAL files. If recovery is needed, we restore the file system backup and then replay from the backed-up WAL files to bring the system to a current state.

#### Особенности

- Нет нужды в идеально консистентном файловом бекапе в качестве начальной точки. Все внутренние несогласованности будут исправлены при проигрывании лога (процесс не принципиально отличается от аварийного восстановления). Достаточно `tar` или чего-то вроде.
- Поскольку мы можем комбинировать бесконечно длинную последовательность файлов WAL для воспроизведения, можно обеспечить непрерывное резервное копирование, просто продолжая архивировать файлы WAL. Это особенно ценно для больших баз данных, где может быть неудобно часто выполнять полное резервное копирование.
- Нет необходимости воспроизводить записи WAL до конца. Мы можем остановить воспроизведение в любой момент и получить согласованный снимок базы данных в том виде, в каком она была на тот момент. Таким образом, этот метод поддерживает восстановление в определенный момент времени: можно восстановить базу данных до ее состояния в любой момент с момента создания базовой резервной копии.
- Если мы постоянно передаем серию файлов WAL на другую машину, на которую был загружен тот же базовый файл резервной копии, у нас есть система "теплого ожидания": в любой момент мы можем запустить вторую машину, и у нее будет почти текущая копия базы данных.
- Работает только для полного резервного копирования и восстановления всего кластера.
- Большие размеры файлов

Заметка из документации:

> pg_dump and pg_dumpall do not produce file-system-level backups and cannot be used as part of a continuous-archiving solution. Such dumps are logical and do not contain enough information to be used by WAL replay.

## VACUUM, VACUUM FULL, ANALYZE

Документация: [postgresql.org/docs/routine-vacuuming](https://www.postgresql.org/docs/current/routine-vacuuming.html)

> PostgreSQL databases require periodic maintenance known as vacuuming. For many installations, it is sufficient to let vacuuming be performed by the autovacuum daemon, which is described in Section 24.1.6. You might need to adjust the autovacuuming parameters described there to obtain best results for your situation. Some database administrators will want to supplement or replace the daemon's activities with manually-managed VACUUM commands, which typically are executed according to a schedule by cron or Task Scheduler scripts.

### VACUUM

> VACUUM — garbage-collect and optionally analyze a database

`VACUUM` приходится регулярно обрабатывать каждую таблицу по нескольким причинам:

- Для восстановления или повторного использования дискового пространства, занятого обновленными или удаленными строками.
- Для обновления статистики, используемой планировщиком запросов PostgreSQL (опционально вызывает `ANALYZE`).
- Для обновления карты видимости, которая ускоряет сканирование только по индексу.
- Для защиты от потери очень старых данных из-за переполнения ID транзакции или multixactID.

`VACUUM` создает значительный объем трафика ввода-вывода, что может привести к снижению производительности других активных сеансов. Существуют параметры конфигурации, которые можно настроить, чтобы уменьшить влияние фоновой очистки на производительность.

> Since transaction IDs have limited size (32 bits) a cluster that runs for a long time (more than 4 billion transactions) would suffer transaction ID wraparound: the XID counter wraps around to zero, and all of a sudden transactions that were in the past appear to be in the future — which means their output become invisible. In short, catastrophic data loss. (Actually the data is still there, but that's cold comfort if you cannot get at it.) To avoid this, it is necessary to vacuum every table in every database at least once every two billion transactions.

### VACUUM FULL

> There are two variants of VACUUM: standard VACUUM and VACUUM FULL. VACUUM FULL can reclaim more disk space but runs much more slowly. Also, the standard form of VACUUM can run in parallel with production database operations. (Commands such as SELECT, INSERT, UPDATE, and DELETE will continue to function normally, though you will not be able to modify the definition of a table with commands such as ALTER TABLE while it is being vacuumed.) VACUUM FULL requires an ACCESS EXCLUSIVE lock on the table it is working on, and therefore cannot be done in parallel with other use of the table. Generally, therefore, administrators should strive to use standard VACUUM and avoid VACUUM FULL.

Стандартный `VACUUM` удаляет пустые строки в таблицах и индексах и отмечает пространство, доступное для повторного использования в будущем. Однако это не вернет пространство операционной системе, за исключением некоторых особых случаев. В отличие от этого, `VACUUM FULL` активно сжимает таблицы, записывая полностью новую версию файла таблицы без пустого пространства. Это минимизирует размер таблицы, но может занять много времени. Это также требует дополнительного места на диске для новой копии таблицы до завершения операции.

> The usual goal of routine vacuuming is to do standard VACUUMs often enough to avoid needing VACUUM FULL. The autovacuum daemon attempts to work this way, and in fact will never issue VACUUM FULL. In this approach, the idea is not to keep tables at their minimum size, but to maintain steady-state usage of disk space: each table occupies space equivalent to its minimum size plus however much space gets used up between vacuum runs. Although VACUUM FULL can be used to shrink a table back to its minimum size and return the disk space to the operating system, there is not much point in this if the table will just grow again in the future. Thus, moderately-frequent standard VACUUM runs are a better approach than infrequent VACUUM FULL runs for maintaining heavily-updated tables.

### ANALYZE

> ANALYZE — collect statistics about a database

`ANALYZE` собирает статистические данные о содержимом таблиц в базе данных и сохраняет результаты в системном каталоге pg_statistic. Впоследствии планировщик запросов использует эти статистические данные для определения наиболее эффективных планов выполнения запросов.

Как и при очистке данных с целью освобождения места, частое обновление статистики более полезно для часто обновляемых таблиц, чем для редко обновляемых. Но даже для часто обновляемой таблицы может не потребоваться обновление статистики, если статистическое распределение данных не сильно меняется.

> It is possible to run ANALYZE on specific tables and even just specific columns of a table, so the flexibility exists to update some statistics more frequently than others if your application requires it. In practice, however, it is usually best to just analyze the entire database, because it is a fast operation. ANALYZE uses a statistically random sampling of the rows of a table rather than reading every single row.

## Блокировки

Документация: [postgresql.org/docs/explicit-locking](https://www.postgresql.org/docs/current/explicit-locking.html)

PostgreSQL предоставляет различные режимы блокировки для управления параллельным доступом к данным в таблицах. Эти режимы могут использоваться в ситуациях, когда `MVCC` не обеспечивает желаемого поведения. Кроме того, большинство команд PostgreSQL автоматически используют соответствующие блокировки, чтобы гарантировать, что задействованные таблицы не будут удалены или изменены несовместимым образом во время выполнения команды.

> To examine a list of the currently outstanding locks in a database server, use the pg_locks system view.

### Table-Level Locks

> You can also acquire any of these locks explicitly with the command LOCK.

Документация команды `LOCK`: [postgresql.org/docs/sql-lock](https://www.postgresql.org/docs/current/sql-lock.html)

> The only real difference between one lock mode and another is the set of lock modes with which each conflicts. Two transactions cannot hold locks of conflicting modes on the same table at the same time. (However, a transaction never conflicts with itself. For example, it might acquire ACCESS EXCLUSIVE lock and later acquire ACCESS SHARE lock on the same table.) Non-conflicting lock modes can be held concurrently by many transactions. Notice in particular that some lock modes are self-conflicting (for example, an ACCESS EXCLUSIVE lock cannot be held by more than one transaction at a time) while others are not self-conflicting (for example, an ACCESS SHARE lock can be held by multiple transactions).

![Conflicting Lock Modes](/assets/Conflicting%20Lock%20Modes.png)

### Row-Level Locks

> Transaction can hold conflicting locks on the same row, even in different subtransactions; but other than that, two transactions can never hold conflicting locks on the same row. Row-level locks do not affect data querying; they block only writers and lockers to the same row. Row-level locks are released at transaction end or during savepoint rollback, just like table-level locks.

![Conflicting Row-Level Locks](assets/Conflicting%20Row-Level%20Locks.png)

### Deadlocks

> The use of explicit locking can increase the likelihood of deadlocks, wherein two (or more) transactions each hold locks that the other wants. For example, if transaction 1 acquires an exclusive lock on table A and then tries to acquire an exclusive lock on table B, while transaction 2 has already exclusive-locked table B and now wants an exclusive lock on table A, then neither one can proceed. PostgreSQL automatically detects deadlock situations and resolves them by aborting one of the transactions involved, allowing the other(s) to complete. (Exactly which transaction will be aborted is difficult to predict and should not be relied upon.)
