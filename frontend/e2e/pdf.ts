import { compose } from './env.ts'

/** A valid one-page PDF showing `text` (hand-written objects + xref, no dependencies). */
export function makePdf(text: string): Buffer {
  const stream = `BT /F1 36 Tf 72 700 Td (${text}) Tj ET`
  const objects = [
    '<< /Type /Catalog /Pages 2 0 R >>',
    '<< /Type /Pages /Kids [3 0 R] /Count 1 >>',
    '<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Contents 4 0 R /Resources << /Font << /F1 5 0 R >> >> >>',
    `<< /Length ${stream.length} >>\nstream\n${stream}\nendstream`,
    '<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>',
  ]
  let out = '%PDF-1.4\n'
  const offsets: number[] = []
  objects.forEach((body, i) => {
    offsets.push(out.length)
    out += `${i + 1} 0 obj\n${body}\nendobj\n`
  })
  const xref = out.length
  out += `xref\n0 ${objects.length + 1}\n0000000000 65535 f \n`
  for (const offset of offsets) out += `${String(offset).padStart(10, '0')} 00000 n \n`
  out += `trailer\n<< /Size ${objects.length + 1} /Root 1 0 R >>\nstartxref\n${xref}\n%%EOF\n`
  return Buffer.from(out, 'latin1')
}

/** Poppler runs inside the portfolio-service container (the host may not have it). */
export function pdfText(pdf: Buffer, firstPage: number, lastPage: number): string {
  return compose(['exec', '-T', 'portfolio-service', 'pdftotext', '-f', String(firstPage), '-l', String(lastPage), '-', '-'], pdf)
}

export function pdfPages(pdf: Buffer): number {
  const out = compose(['exec', '-T', 'portfolio-service', 'pdfinfo', '-'], pdf)
  return Number(/Pages:\s+(\d+)/.exec(out)?.[1] ?? 0)
}
