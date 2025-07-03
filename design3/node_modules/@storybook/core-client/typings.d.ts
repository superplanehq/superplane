declare class AnsiToHtml {
  constructor(options: { escapeHtml: boolean });

  toHtml: (ansi: string) => string;
}
