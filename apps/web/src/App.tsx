import { FormEvent, useMemo, useState } from "react";
import {
  ApiResult,
  createTransaction,
  validateTransaction
} from "./api";
import { TransactionFormState, toTransactionPayload } from "./transaction";

type RecentEvent = {
  id: string;
  institution: string;
  amount: string;
  status: "valid" | "created" | "blocked";
  time: string;
};

const initialForm: TransactionFormState = {
  id: "txn-2026-0001",
  institutionId: "minfin",
  fiscalYear: "2026",
  amountMinor: "125000",
  currency: "USD",
  transactionDate: "2026-06-28",
  description: "Road maintenance payment"
};

const initialEvents: RecentEvent[] = [
  { id: "txn-2026-0001", institution: "minfin", amount: "USD 1,250.00", status: "valid", time: "Now" },
  { id: "txn-2026-0000", institution: "transport", amount: "USD 980.00", status: "created", time: "12m" },
  { id: "txn-2025-0942", institution: "health", amount: "USD 4,500.00", status: "blocked", time: "42m" }
];

export function App() {
  const [form, setForm] = useState<TransactionFormState>(initialForm);
  const [result, setResult] = useState<ApiResult | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [events, setEvents] = useState<RecentEvent[]>(initialEvents);

  const transaction = useMemo(() => toTransactionPayload(form), [form]);

  async function submitTransaction(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    await runAction("validate");
  }

  async function runAction(action: "validate" | "create") {
    setIsSubmitting(true);
    const response =
      action === "validate"
        ? await validateTransaction(transaction)
        : await createTransaction(transaction);

    setResult(response);
    setEvents((current) => [
      {
        id: transaction.id || "unsaved",
        institution: transaction.institutionId || "unknown",
        amount: `${transaction.currency || "USD"} ${(transaction.amountMinor / 100).toLocaleString(undefined, {
          minimumFractionDigits: 2,
          maximumFractionDigits: 2
        })}`,
        status: response.ok ? (action === "create" ? "created" : "valid") : "blocked",
        time: "Now"
      },
      ...current.slice(0, 4)
    ]);
    setIsSubmitting(false);
  }

  return (
    <div className="app-shell">
      <aside className="sidebar" aria-label="Primary navigation">
        <div className="brand">
          <span className="brand-mark">OT</span>
          <span>OpenTreasury</span>
        </div>
        <nav className="nav-list">
          {["Overview", "Transactions", "Validation", "Institutions", "Audit Trail"].map((item) => (
            <a className={item === "Validation" ? "active" : ""} href={`#${item.toLowerCase().replaceAll(" ", "-")}`} key={item}>
              {item}
            </a>
          ))}
        </nav>
      </aside>

      <main className="workspace">
        <header className="topbar">
          <div>
            <h1>Treasury validation</h1>
            <p>Review transaction payloads before persistence and public audit publication.</p>
          </div>
          <div className="status-cluster" aria-label="System status">
            <div className="status-item">
              <span>Environment</span>
              <strong><span className="status-dot" />Local</strong>
            </div>
            <div className="status-item">
              <span>API Health</span>
              <strong><span className="status-dot" />Ready</strong>
            </div>
            <div className="status-item">
              <span>Contract</span>
              <strong>OpenAPI 3.1</strong>
            </div>
          </div>
        </header>

        <section className="content-grid">
          <form className="panel transaction-form" onSubmit={submitTransaction}>
            <div className="panel-heading">
              <h2>Transaction payload</h2>
              <span className="panel-code">POST /v1/transactions</span>
            </div>

            <div className="form-grid">
              <Field label="Transaction ID" value={form.id} onChange={(id) => setForm({ ...form, id })} />
              <Field label="Institution ID" value={form.institutionId} onChange={(institutionId) => setForm({ ...form, institutionId })} />
              <Field label="Fiscal Year" value={form.fiscalYear} onChange={(fiscalYear) => setForm({ ...form, fiscalYear })} inputMode="numeric" />
              <Field label="Amount Minor" value={form.amountMinor} onChange={(amountMinor) => setForm({ ...form, amountMinor })} inputMode="numeric" />
              <Field label="Currency" value={form.currency} onChange={(currency) => setForm({ ...form, currency })} maxLength={3} />
              <Field label="Transaction Date" value={form.transactionDate} onChange={(transactionDate) => setForm({ ...form, transactionDate })} type="date" />
            </div>

            <label className="field field-full">
              <span>Description</span>
              <textarea value={form.description} onChange={(event) => setForm({ ...form, description: event.target.value })} rows={4} />
            </label>

            <div className="form-actions">
              <button type="submit" disabled={isSubmitting}>
                Validate Transaction
              </button>
              <button type="button" className="secondary" disabled={isSubmitting} onClick={() => void runAction("create")}>
                Create Transaction
              </button>
            </div>
          </form>

          <section className="panel result-panel" aria-label="Validation result">
            <div className="panel-heading">
              <h2>Validation result</h2>
              <StatusBadge result={result} />
            </div>
            <pre className="response-preview">{formatResult(result)}</pre>

            <div className="table-heading">
              <h2>Recent activity</h2>
              <span>{events.length} records</span>
            </div>
            <table>
              <thead>
                <tr>
                  <th>Transaction</th>
                  <th>Institution</th>
                  <th>Amount</th>
                  <th>Status</th>
                  <th>Age</th>
                </tr>
              </thead>
              <tbody>
                {events.map((event, index) => (
                  <tr key={`${event.id}-${event.time}-${index}`}>
                    <td>{event.id}</td>
                    <td>{event.institution}</td>
                    <td>{event.amount}</td>
                    <td>
                      <span className={`row-status ${event.status}`}>{event.status}</span>
                    </td>
                    <td>{event.time}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </section>
        </section>
      </main>
    </div>
  );
}

function Field(props: {
  label: string;
  value: string;
  onChange: (value: string) => void;
  type?: string;
  inputMode?: "numeric";
  maxLength?: number;
}) {
  return (
    <label className="field">
      <span>{props.label}</span>
      <input
        type={props.type ?? "text"}
        value={props.value}
        inputMode={props.inputMode}
        maxLength={props.maxLength}
        onChange={(event) => props.onChange(event.target.value)}
      />
    </label>
  );
}

function StatusBadge({ result }: { result: ApiResult | null }) {
  if (!result) {
    return <span className="badge neutral">Waiting</span>;
  }

  return result.ok ? <span className="badge ok">Accepted</span> : <span className="badge blocked">Blocked</span>;
}

function formatResult(result: ApiResult | null) {
  if (!result) {
    return JSON.stringify(
      {
        status: "ready",
        message: "Submit a transaction payload to validate it against the core API."
      },
      null,
      2
    );
  }

  return JSON.stringify(result, null, 2);
}
