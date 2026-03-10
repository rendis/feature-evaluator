export type Operator =
  | '=='
  | '!='
  | '>'
  | '>='
  | '<'
  | '<='
  | 'contains'
  | 'startsWith'
  | 'endsWith'
  | 'matches'
  | 'in'
  | 'not in';

export type Connector = 'and' | 'or';

export interface Condition {
  field: string;
  operator: Operator;
  value: string;
}

export interface ConditionGroup {
  connector: Connector;
  conditions: (Condition | ConditionGroup)[];
}

export function isConditionGroup(item: Condition | ConditionGroup): item is ConditionGroup {
  return 'conditions' in item;
}

export function emptyCondition(): Condition {
  return { field: '', operator: '==', value: '' };
}

export function emptyGroup(connector: Connector = 'and'): ConditionGroup {
  return { connector, conditions: [emptyCondition()] };
}

/** Serialize a condition group to an expr-lang expression string */
export function serializeExpression(group: ConditionGroup): string {
  const parts = group.conditions.map((item) => {
    if (isConditionGroup(item)) {
      return `(${serializeExpression(item)})`;
    }
    const { field, operator, value } = item;
    if (operator === 'in' || operator === 'not in') {
      return `${field} ${operator} [${value}]`;
    }
    if (operator === 'contains' || operator === 'startsWith' || operator === 'endsWith') {
      return `${field} ${operator} "${value}"`;
    }
    if (operator === 'matches') {
      return `${field} matches "${value}"`;
    }
    const numericValue = Number(value);
    const isNumeric = !isNaN(numericValue) && value.trim() !== '';
    const isBool = value === 'true' || value === 'false';
    const formatted = isNumeric || isBool ? value : `"${value}"`;
    return `${field} ${operator} ${formatted}`;
  });

  const conn = group.connector === 'and' ? ' && ' : ' || ';
  return parts.join(conn);
}
