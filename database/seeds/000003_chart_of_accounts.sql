-- Reference chart of accounts, aligned with the IMF GFSM 2014 economic
-- classification. Deployments adapt or replace this via CoA import.
INSERT INTO chart_of_accounts (version, status, description) VALUES
  (1, 'ACTIVE', 'Reference chart of accounts (GFSM 2014 economic classification)')
ON CONFLICT (version) DO NOTHING;

INSERT INTO coa_accounts (version, code, name, account_type, parent_code, gfsm_code) VALUES
  (1, '1',   'Revenue',                              'REVENUE',   NULL, '1'),
  (1, '11',  'Taxes',                                'REVENUE',   '1',  '11'),
  (1, '111', 'Taxes on income, profits, and capital gains', 'REVENUE', '11', '111'),
  (1, '114', 'Taxes on goods and services',          'REVENUE',   '11', '114'),
  (1, '115', 'Taxes on international trade',         'REVENUE',   '11', '115'),
  (1, '12',  'Social contributions',                 'REVENUE',   '1',  '12'),
  (1, '13',  'Grants',                               'REVENUE',   '1',  '13'),
  (1, '14',  'Other revenue',                        'REVENUE',   '1',  '14'),
  (1, '2',   'Expense',                              'EXPENSE',   NULL, '2'),
  (1, '21',  'Compensation of employees',            'EXPENSE',   '2',  '21'),
  (1, '22',  'Use of goods and services',            'EXPENSE',   '2',  '22'),
  (1, '24',  'Interest',                             'EXPENSE',   '2',  '24'),
  (1, '25',  'Subsidies',                            'EXPENSE',   '2',  '25'),
  (1, '26',  'Grants',                               'EXPENSE',   '2',  '26'),
  (1, '27',  'Social benefits',                      'EXPENSE',   '2',  '27'),
  (1, '28',  'Other expense',                        'EXPENSE',   '2',  '28'),
  (1, '31',  'Nonfinancial assets',                  'ASSET',     NULL, '31'),
  (1, '311', 'Fixed assets',                         'ASSET',     '31', '311'),
  (1, '62',  'Financial assets',                     'ASSET',     NULL, '62'),
  (1, '6202','Currency and deposits',                'ASSET',     '62', '6202'),
  (1, '6203','Debt securities',                      'ASSET',     '62', '6203'),
  (1, '63',  'Liabilities',                          'LIABILITY', NULL, '63'),
  (1, '6303','Debt securities issued',               'LIABILITY', '63', '6303'),
  (1, '6304','Loans',                                'LIABILITY', '63', '6304'),
  (1, '6',   'Net worth',                            'NET_WORTH', NULL, '6')
ON CONFLICT (version, code) DO NOTHING;
