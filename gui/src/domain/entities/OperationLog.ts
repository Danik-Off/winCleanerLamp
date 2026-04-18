/**
 * Value Object: LogLevel
 */
export enum LogLevel {
  INFO = 'info',
  WARN = 'warn',
  ERROR = 'error',
  SUCCESS = 'success',
  PROGRESS = 'progress'
}

/**
 * Domain Entity: LogEntry
 * Single log line from operation
 */
export class LogEntry {
  constructor(
    public readonly timestamp: Date,
    public readonly level: LogLevel,
    public readonly message: string,
    public readonly data?: unknown
  ) {}

  static info(message: string): LogEntry {
    return new LogEntry(new Date(), LogLevel.INFO, message);
  }

  static warn(message: string): LogEntry {
    return new LogEntry(new Date(), LogLevel.WARN, message);
  }

  static error(message: string, data?: unknown): LogEntry {
    return new LogEntry(new Date(), LogLevel.ERROR, message, data);
  }

  static success(message: string): LogEntry {
    return new LogEntry(new Date(), LogLevel.SUCCESS, message);
  }

  static progress(message: string): LogEntry {
    return new LogEntry(new Date(), LogLevel.PROGRESS, message);
  }

  toString(): string {
    const time = this.timestamp.toLocaleTimeString();
    return `[${time}] [${this.level.toUpperCase()}] ${this.message}`;
  }
}

/**
 * Domain Entity: OperationLog
 * Aggregate of log entries with filtering capabilities
 */
export class OperationLog {
  constructor(
    private readonly entries: ReadonlyArray<LogEntry>
  ) {}

  static empty(): OperationLog {
    return new OperationLog([]);
  }

  add(entry: LogEntry): OperationLog {
    return new OperationLog([...this.entries, entry]);
  }

  addRaw(text: string): OperationLog {
    const lines = text.split('\n').filter(l => l.trim());
    const newEntries = lines.map(line => LogEntry.info(line));
    return new OperationLog([...this.entries, ...newEntries]);
  }

  get all(): LogEntry[] {
    return [...this.entries];
  }

  get errors(): LogEntry[] {
    return this.entries.filter(e => e.level === LogLevel.ERROR);
  }

  get hasErrors(): boolean {
    return this.errors.length > 0;
  }

  get lastEntry(): LogEntry | undefined {
    return this.entries[this.entries.length - 1];
  }

  get count(): number {
    return this.entries.length;
  }

  filterByLevel(level: LogLevel): LogEntry[] {
    return this.entries.filter(e => e.level === level);
  }

  toString(): string {
    return this.entries.map(e => e.message).join('\n');
  }
}
