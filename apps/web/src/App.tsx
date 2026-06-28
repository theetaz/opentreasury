import { FormEvent, useCallback, useEffect, useMemo, useState } from "react";
import {
  ApiResult,
  createTransaction,
  listTransactions,
  validateTransaction
} from "./api";
import type { TransactionListFilter } from "./api";
import {
  HistoryFilterForm,
  RecentEvent,
  toRecentEvent,
  toTransactionListFilter
} from "./history";
import { TransactionFormState, toTransactionPayload } from "./transaction";

const initialForm: TransactionFormState = {
  id: "txn-2026-0001",
  institutionId: "minfin",
  fiscalYear: "2026",
  amountMinor: "125000",
  currency: "USD",
  transactionDate: "2026-06-28",
  description: "Road maintenance payment"
};

const initialHistoryFilter: HistoryFilterForm = {
  institutionId: "",
  fiscalYear: "",
  limit: "5"
};

export function App() {
  const [form, setForm] = useState<TransactionFormState>(initialForm);
  const [historyFilter, setHistoryFilter] = useState<HistoryFilterForm>(initialHistoryFilter);
  const [appliedHistoryFilter, setAppliedHistoryFilter] = useState<TransactionListFilter>(() => toTransactionListFilter(initialHistoryFilter));
  const [result, setResult] = useState<ApiResult | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [events, setEvents] = useState<RecentEvent[]>([]);
  const [historyStatus, setHistoryStatus] = useState<"loading" | "ready" | "error">("loading");
  const [historyError, setHistoryError] = useState("");

  const transaction = useMemo(() => toTransactionPayload(form), [form]);

  const loadHistory = useCallback(async (filter: TransactionListFilter) => {
    setHistoryStatus("loading");
    const response = await listTransactions(filter);
    if (response.ok) {
      setEvents(response.transactions.map(toRecentEvent));
      setHistoryStatus("ready");
      setHistoryError("");
      return;
    }

    setHistoryStatus("error");
    setHistoryError(response.error);
  }, []);

  useEffect(() => {
    void loadHistory(appliedHistoryFilter);
  }, [appliedHistoryFilter, loadHistory]);

  async function submitTransaction(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    await runAction("validate");
  }

  async function submitHistoryFilter(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setAppliedHistoryFilter(toTransactionListFilter(historyFilter));
  }

  async function runAction(action: "validate" | "create") {
    setIsSubmitting(true);
    const response =
      action === "validate"
        ? await validateTransaction(transaction)
        : await createTransaction(transaction);

    setResult(response);
    if (response.ok && action === "create") {
      setEvents((current) => [toRecentEvent(transaction), ...current.filter((event) => event.id !== transaction.id)].slice(0, 5));
    }
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
              <button className="text-button" type="button" onClick={() => void loadHistory(appliedHistoryFilter)}>
                Refresh
              </button>
            </div>
            <form className="history-filters" onSubmit={submitHistoryFilter} aria-label="Transaction history filters">
              <Field
                label="Institution"
                value={historyFilter.institutionId}
                onChange={(institutionId) => setHistoryFilter({ ...historyFilter, institutionId })}
              />
              <Field
                label="Fiscal Year"
                value={historyFilter.fiscalYear}
                onChange={(fiscalYear) => setHistoryFilter({ ...historyFilter, fiscalYear })}
                inputMode="numeric"
              />
              <Field
                label="Limit"
                value={historyFilter.limit}
                onChange={(limit) => setHistoryFilter({ ...historyFilter, limit })}
                inputMode="numeric"
              />
              <button className="secondary" type="submit">
                Apply
              </button>
            </form>
            <table>
              <thead>
                <tr>
                  <th>Transaction</th>
                  <th>Institution</th>
                  <th>Date</th>
                  <th>Amount</th>
                  <th>Status</th>
                </tr>
              </thead>
              <tbody>
                <HistoryRows events={events} status={historyStatus} error={historyError} />
              </tbody>
            </table>
          </section>
        </section>
      </main>
    </div>
  );
}

function HistoryRows(props: {
  events: RecentEvent[];
  status: "loading" | "ready" | "error";
  error: string;
}) {
  if (props.status === "loading") {
    return (
      <tr>
        <td colSpan={5}>Loading transaction history...</td>
      </tr>
    );
  }

  if (props.status === "error") {
    return (
      <tr>
        <td colSpan={5}>{props.error}</td>
      </tr>
    );
  }

  if (props.events.length === 0) {
    return (
      <tr>
        <td colSpan={5}>No transactions found.</td>
      </tr>
    );
  }

  return props.events.map((event) => (
    <tr key={`${event.id}-${event.date}`}>
      <td>{event.id}</td>
      <td>{event.institution}</td>
      <td>{event.date}</td>
      <td>{event.amount}</td>
      <td>
        <span className={`row-status ${event.status}`}>{event.status}</span>
      </td>
    </tr>
  ));
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
