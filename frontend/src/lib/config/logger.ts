// So far let's keep it simple and log to console

/* eslint-disable @typescript-eslint/no-explicit-any */

interface Logger {
  debug(...args: any[]): void;
  info(...args: any[]): void;
  warn(...args: any[]): void;
  error(...args: any[]): void;
}

class ConsoleLogger implements Logger {
  debug(...args: any[]): void {
    console.debug(...args);
  }
  info(...args: any[]): void {
    console.info(...args);
  }
  warn(...args: any[]): void {
    console.warn(...args);
  }
  error(...args: any[]): void {
    console.error(...args);
  }
}

export const logger: Logger = new ConsoleLogger();