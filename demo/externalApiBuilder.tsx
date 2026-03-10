import {
  AlertCircle,
  AlignLeft,
  ArrowLeft,
  ArrowRight,
  Braces,
  Check,
  Code,
  Edit3,
  FileJson,
  GitCommit,
  Globe,
  ListTree,
  Play,
  Plus,
  Settings,
  Terminal,
  Trash2,
  X,
} from "lucide-react";
import { useEffect, useMemo, useRef, useState } from "react";

// --- FUNCIONES AUXILIARES PARA JSON <-> VISUAL ---

const visualNodesToJson = (nodes, parentType = "object") => {
  if (parentType === "array") {
    return nodes.map((n) => {
      if (n.type === "object")
        return visualNodesToJson(n.children || [], "object");
      if (n.type === "array")
        return visualNodesToJson(n.children || [], "array");
      let v = n.value;
      if (n.type === "number" && !isNaN(Number(v)) && v !== "")
        return Number(v);
      if (n.type === "boolean") return v === "true";
      return v;
    });
  } else {
    const obj = {};
    nodes.forEach((n) => {
      if (!n.key) return;
      if (n.type === "object")
        obj[n.key] = visualNodesToJson(n.children || [], "object");
      else if (n.type === "array")
        obj[n.key] = visualNodesToJson(n.children || [], "array");
      else {
        let v = n.value;
        if (n.type === "number" && !isNaN(Number(v)) && v !== "") v = Number(v);
        else if (n.type === "boolean") v = v === "true";
        obj[n.key] = v;
      }
    });
    return obj;
  }
};

const jsonToVisualNodes = (data, prefix = "root") => {
  if (Array.isArray(data)) {
    return data.map((val, idx) => {
      const id = `${prefix}-${idx}`;
      const isObj =
        val !== null && typeof val === "object" && !Array.isArray(val);
      const isArr = Array.isArray(val);
      if (isObj)
        return {
          id,
          key: "",
          type: "object",
          value: "",
          children: jsonToVisualNodes(val, id),
        };
      if (isArr)
        return {
          id,
          key: "",
          type: "array",
          value: "",
          children: jsonToVisualNodes(val, id),
        };
      let type = typeof val;
      if (!["string", "number", "boolean"].includes(type)) type = "string";
      return { id, key: "", type, value: String(val), children: [] };
    });
  } else if (data !== null && typeof data === "object") {
    return Object.keys(data).map((key, idx) => {
      const val = data[key];
      const id = `${prefix}-${idx}`;
      const isObj =
        val !== null && typeof val === "object" && !Array.isArray(val);
      const isArr = Array.isArray(val);
      if (isObj)
        return {
          id,
          key,
          type: "object",
          value: "",
          children: jsonToVisualNodes(val, id),
        };
      if (isArr)
        return {
          id,
          key,
          type: "array",
          value: "",
          children: jsonToVisualNodes(val, id),
        };
      let type = typeof val;
      if (!["string", "number", "boolean"].includes(type)) type = "string";
      return { id, key, type, value: String(val), children: [] };
    });
  }
  return [];
};

// --- COMPONENTES AUXILIARES PARA RESALTADO (OVERLAY) ---

const renderHighlightedText = (text, defaultClass = "text-slate-300") => {
  if (!text) return null;
  const parts = text.split(/(\{\{[^}]*(?:\}\}|$))/g);
  return parts.map((part, index) => {
    if (part.startsWith("{{")) {
      const isClosed = part.endsWith("}}");
      const innerText = isClosed ? part.slice(2, -2) : part.slice(2);
      const isSnakeCase =
        /^[a-z0-9_]*$/.test(innerText) && innerText.length > 0;

      const bgClass = isSnakeCase
        ? "bg-indigo-500/40 text-indigo-200"
        : "bg-rose-500/40 text-rose-200 underline decoration-rose-400 decoration-wavy decoration-1 underline-offset-4";

      return (
        <span key={index} className={`${bgClass} rounded-sm`}>
          {part}
        </span>
      );
    }
    return (
      <span key={index} className={defaultClass}>
        {part}
      </span>
    );
  });
};

const HighlightedInput = ({ value, onChange, placeholder, className }) => {
  const inputRef = useRef(null);
  const backdropRef = useRef(null);

  const handleScroll = (e) => {
    if (backdropRef.current)
      backdropRef.current.scrollLeft = e.target.scrollLeft;
  };

  const handleKeyDown = (e) => {
    const input = e.target;
    const start = input.selectionStart;
    const end = input.selectionEnd;

    if (e.key === "{") {
      if (start > 0 && value.charAt(start - 1) === "{") {
        e.preventDefault();
        const newVal = value.substring(0, start) + "{}}" + value.substring(end);
        onChange(newVal);
        setTimeout(() => {
          input.focus();
          input.setSelectionRange(start + 1, start + 1);
        }, 0);
      }
    }
  };

  return (
    <div className={`relative overflow-hidden ${className}`}>
      <div
        ref={backdropRef}
        className="absolute inset-0 px-4 py-2 font-mono text-sm whitespace-pre overflow-hidden pointer-events-none"
        aria-hidden="true"
      >
        {renderHighlightedText(value) || (
          <span className="text-slate-500">{placeholder}</span>
        )}
      </div>
      <input
        ref={inputRef}
        type="text"
        value={value}
        onChange={(e) => onChange(e.target.value)}
        onKeyDown={handleKeyDown}
        onScroll={handleScroll}
        className="absolute inset-0 w-full h-full bg-transparent px-4 py-2 font-mono text-sm text-transparent caret-white focus:outline-none z-10 selection:bg-indigo-500/40 selection:text-transparent"
        spellCheck="false"
      />
    </div>
  );
};

const HighlightedTextarea = ({ value, onChange, placeholder, className }) => {
  const inputRef = useRef(null);
  const backdropRef = useRef(null);

  const handleScroll = (e) => {
    if (backdropRef.current) backdropRef.current.scrollTop = e.target.scrollTop;
  };

  const handleKeyDown = (e) => {
    const input = e.target;
    const start = input.selectionStart;
    const end = input.selectionEnd;

    if (e.key === "{") {
      if (start > 0 && value.charAt(start - 1) === "{") {
        e.preventDefault();
        const newVal = value.substring(0, start) + "{}}" + value.substring(end);
        onChange(newVal);
        setTimeout(() => {
          input.focus();
          input.setSelectionRange(start + 1, start + 1);
        }, 0);
      }
    }
  };

  return (
    <div className={`relative overflow-hidden ${className}`}>
      <div
        ref={backdropRef}
        className="absolute inset-0 p-4 font-mono text-sm whitespace-pre-wrap break-words overflow-hidden pointer-events-none"
        aria-hidden="true"
      >
        {renderHighlightedText(value)}
        {!value && <span className="text-slate-500">{placeholder}</span>}
      </div>
      <textarea
        ref={inputRef}
        value={value}
        onChange={(e) => onChange(e.target.value)}
        onKeyDown={handleKeyDown}
        onScroll={handleScroll}
        className="absolute inset-0 w-full h-full bg-transparent p-4 font-mono text-sm text-transparent caret-white resize-none outline-none z-10 selection:bg-indigo-500/40 selection:text-transparent"
        spellCheck="false"
      />
    </div>
  );
};

// --- VALIDADOR Y RESALTADOR SEMÁNTICO DE EXPRESIONES ---
const renderExpressionText = (text, validTokens) => {
  if (!text) return null;

  // Divide la expresión preservando operadores, espacios, strings y variables
  const regex =
    /(\bresponse(?:\[\d+\]|\.[a-zA-Z0-9_]+)*\b|\{\{[a-zA-Z0-9_]+\}\}|"[^"]*"|'[^']*'|-?\b\d+(?:\.\d+)?\b|[a-zA-Z_][a-zA-Z0-9_]*|[=!><]=?|&&|\|\||[\s()[\]{}]+)/g;
  const parts = text.split(regex).filter(Boolean);

  const reservedWords = new Set([
    "and",
    "or",
    "not",
    "true",
    "false",
    "null",
    "empty",
    "contains",
    "startsWith",
    "endsWith",
    "length",
    "in",
    "matches",
  ]);

  return parts.map((part, index) => {
    // 1. Espacios y puntuación básica
    if (/^[\s()[\]{}]+$/.test(part)) return <span key={index}>{part}</span>;

    // 2. Variables Paramétricas {{var}}
    if (part.startsWith("{{") && part.endsWith("}}")) {
      if (validTokens.includes(part)) {
        return (
          <span
            key={index}
            className="text-indigo-300 bg-indigo-500/20 px-0.5 rounded"
          >
            {part}
          </span>
        );
      }
      return (
        <span
          key={index}
          className="text-rose-400 bg-rose-500/20 underline decoration-wavy px-0.5 rounded"
          title="Variable no definida en el sistema"
        >
          {part}
        </span>
      );
    }

    // 3. Rutas del Schema (ej. response.results[0].id)
    if (part.startsWith("response")) {
      // Limpiamos los índices de arreglo para compararlos con el schema (ej. results[0].id -> results.id)
      const cleanPath = part.replace(/\[\d+\]/g, "");
      // Comprobamos si la ruta existe en el schema (o al menos la base de la ruta)
      const isValid = validTokens.some((t) => t.startsWith(cleanPath));
      if (isValid) {
        return (
          <span key={index} className="text-emerald-400">
            {part}
          </span>
        );
      }
      return (
        <span
          key={index}
          className="text-rose-400 bg-rose-500/20 underline decoration-wavy px-0.5 rounded"
          title="Esta ruta no se encontró en el Response Schema"
        >
          {part}
        </span>
      );
    }

    // 4. Palabras reservadas del motor
    if (reservedWords.has(part)) {
      return (
        <span key={index} className="text-purple-400 font-bold">
          {part}
        </span>
      );
    }

    // 5. Strings ("algo")
    if (
      (part.startsWith('"') && part.endsWith('"')) ||
      (part.startsWith("'") && part.endsWith("'"))
    ) {
      return (
        <span key={index} className="text-amber-300">
          {part}
        </span>
      );
    }

    // 6. Números
    if (/^-?\d+(\.\d+)?$/.test(part)) {
      return (
        <span key={index} className="text-blue-300">
          {part}
        </span>
      );
    }

    // 7. Operadores Lógicos/Matemáticos
    if (/^[=!><]=?|&&|\|\|$/.test(part)) {
      return (
        <span key={index} className="text-pink-400">
          {part}
        </span>
      );
    }

    // 8. Fallback: Palabra desconocida (Error Semántico)
    return (
      <span
        key={index}
        className="text-rose-400 bg-rose-500/20 underline decoration-wavy px-0.5 rounded"
        title="Token desconocido o sintaxis inválida"
      >
        {part}
      </span>
    );
  });
};

// --- COMPONENTE DE AUTOCOMPLETADO PARA EXPRESIONES ---
const ExpressionEditor = ({ value, onChange, tokens }) => {
  const [suggestions, setSuggestions] = useState([]);
  const [activeIdx, setActiveIdx] = useState(0);
  const [cursorPos, setCursorPos] = useState(0);
  const textareaRef = useRef(null);

  const handleInput = (e) => {
    const val = e.target.value;
    onChange(val);
    const cursor = e.target.selectionStart;
    setCursorPos(cursor);

    const textBefore = val.slice(0, cursor);
    const match = textBefore.match(/([a-zA-Z0-9_.]+|\{\{[a-zA-Z0-9_]*)$/);
    if (match) {
      const word = match[0];
      const matches = tokens
        .filter((t) => t.startsWith(word) && t !== word)
        .slice(0, 8);
      setSuggestions(matches);
      setActiveIdx(0);
    } else {
      setSuggestions([]);
    }
  };

  const handleKeyDown = (e) => {
    const start = e.target.selectionStart;
    const end = e.target.selectionEnd;

    if (e.key === "{") {
      if (start > 0 && value.charAt(start - 1) === "{") {
        e.preventDefault();
        const newVal = value.substring(0, start) + "{}}" + value.substring(end);
        onChange(newVal);
        setTimeout(() => {
          if (textareaRef.current) {
            textareaRef.current.focus();
            textareaRef.current.setSelectionRange(start + 1, start + 1);
          }
        }, 0);
        return;
      }
    }

    if (suggestions.length > 0) {
      if (e.key === "ArrowDown") {
        e.preventDefault();
        setActiveIdx((prev) => (prev + 1) % suggestions.length);
      } else if (e.key === "ArrowUp") {
        e.preventDefault();
        setActiveIdx(
          (prev) => (prev - 1 + suggestions.length) % suggestions.length,
        );
      } else if (e.key === "Tab" || e.key === "Enter") {
        e.preventDefault();
        insertSuggestion(suggestions[activeIdx]);
      } else if (e.key === "Escape") {
        setSuggestions([]);
      }
    }
  };

  const insertSuggestion = (suggestion) => {
    const textBefore = value.slice(0, cursorPos);
    const textAfter = value.slice(cursorPos);
    const match = textBefore.match(/([a-zA-Z0-9_.]+|\{\{[a-zA-Z0-9_]*)$/);
    if (match) {
      const word = match[0];
      const prefix = textBefore.slice(0, -word.length);
      const newVal = prefix + suggestion + textAfter;
      onChange(newVal);
      setSuggestions([]);
      setTimeout(() => {
        if (textareaRef.current) {
          textareaRef.current.focus();
          const newPos = prefix.length + suggestion.length;
          textareaRef.current.setSelectionRange(newPos, newPos);
        }
      }, 0);
    }
  };

  return (
    <div className="relative w-full h-full min-h-[160px] flex">
      <div
        className="absolute inset-0 p-3 font-mono text-sm whitespace-pre-wrap break-words overflow-hidden pointer-events-none"
        aria-hidden="true"
      >
        {renderExpressionText(value, tokens)}
        {!value && (
          <span className="text-slate-500">
            Ej: response.results.length &gt; 0 and response.results[0].id =={" "}
            {"{{"}campus_code{"}}"}
          </span>
        )}
      </div>
      <textarea
        ref={textareaRef}
        value={value}
        onChange={handleInput}
        onKeyDown={handleKeyDown}
        onClick={(e) => {
          setCursorPos(e.target.selectionStart);
          setSuggestions([]);
        }}
        className="w-full h-full flex-1 bg-slate-900 border border-slate-700 rounded-lg p-3 text-transparent font-mono text-sm focus:border-indigo-500 caret-white selection:bg-indigo-500/40 selection:text-transparent resize-none"
        spellCheck="false"
      />
      {suggestions.length > 0 && (
        <ul className="absolute z-50 bg-slate-800 border border-slate-600 rounded-lg shadow-2xl mt-1 max-h-48 overflow-y-auto min-w-[300px]">
          {suggestions.map((s, idx) => (
            <li
              key={s}
              onClick={() => insertSuggestion(s)}
              className={`px-3 py-2 text-xs font-mono cursor-pointer flex items-center gap-2 ${idx === activeIdx ? "bg-indigo-600 text-white" : "text-slate-300 hover:bg-slate-700"}`}
            >
              <Code className="w-3 h-3 opacity-50" /> {s}
            </li>
          ))}
        </ul>
      )}
    </div>
  );
};

// --- COMPONENTE GENÉRICO EDITOR VISUAL (ÁRBOL) ---
const VisualTreeEditor = ({
  nodes,
  onChange,
  rootType,
  allowParams = true,
}) => {
  const updateVisualNode = (currentNodes, id, field, value) => {
    return currentNodes.map((n) => {
      if (n.id === id) return { ...n, [field]: value };
      if (n.children && n.children.length > 0)
        return {
          ...n,
          children: updateVisualNode(n.children, id, field, value),
        };
      return n;
    });
  };

  const removeVisualNode = (currentNodes, id) => {
    return currentNodes
      .filter((n) => n.id !== id)
      .map((n) => ({
        ...n,
        children: n.children ? removeVisualNode(n.children, id) : [],
      }));
  };

  const addVisualNode = (currentNodes, parentId = null) => {
    const newNode = {
      id: Date.now().toString(),
      key: "",
      type: "string",
      value: "",
      children: [],
    };
    if (!parentId) return [...currentNodes, newNode];
    return currentNodes.map((n) => {
      if (n.id === parentId)
        return { ...n, children: [...(n.children || []), newNode] };
      if (n.children && n.children.length > 0)
        return { ...n, children: addVisualNode(n.children, parentId) };
      return n;
    });
  };

  const toggleParamBrackets = (currentNodes, id) => {
    return currentNodes.map((n) => {
      if (n.id === id) {
        let val = String(n.value || "");
        if (val.startsWith("{{") && val.endsWith("}}")) val = val.slice(2, -2);
        else val = `{{${val}}}`;
        return { ...n, value: val };
      }
      if (n.children && n.children.length > 0)
        return { ...n, children: toggleParamBrackets(n.children, id) };
      return n;
    });
  };

  const renderTree = (currentNodes, level = 0, parentType = "object") => {
    return currentNodes.map((row, idx) => (
      <div
        key={row.id}
        className={`flex flex-col gap-1 ${level > 0 ? "ml-6 mt-2 border-l-2 border-slate-700/50 pl-4" : "mt-2"}`}
      >
        <div className="flex gap-2 items-center bg-slate-900/50 p-2 rounded border border-slate-700/50 focus-within:border-indigo-500/50 transition-colors">
          {parentType === "array" ? (
            <div className="w-1/3 px-2 py-1.5 text-xs text-slate-500 font-mono flex justify-center items-center bg-slate-900/50 border border-slate-800 rounded">
              [ {idx} ]
            </div>
          ) : (
            <input
              type="text"
              placeholder="Key"
              value={row.key}
              onChange={(e) =>
                onChange(updateVisualNode(nodes, row.id, "key", e.target.value))
              }
              className="w-1/3 bg-slate-900 border border-slate-700 rounded px-2 py-1.5 text-xs text-white focus:outline-none focus:border-indigo-500"
            />
          )}

          <select
            value={row.type}
            onChange={(e) =>
              onChange(updateVisualNode(nodes, row.id, "type", e.target.value))
            }
            className="w-24 bg-slate-900 border border-slate-700 rounded px-2 py-1.5 text-xs text-slate-300 focus:outline-none"
          >
            <option value="string">String</option>
            <option value="number">Number</option>
            <option value="boolean">Boolean</option>
            <option value="object">Object</option>
            <option value="array">Array</option>
          </select>

          {row.type !== "object" && row.type !== "array" ? (
            <div className="flex-1 relative flex items-center gap-1">
              <input
                type="text"
                placeholder="Valor Ejemplo"
                value={row.value}
                onChange={(e) =>
                  onChange(
                    updateVisualNode(nodes, row.id, "value", e.target.value),
                  )
                }
                className={`flex-1 bg-slate-900 border rounded px-2 py-1.5 text-xs font-mono focus:outline-none ${
                  String(row.value).includes("{{")
                    ? "text-indigo-300 border-indigo-500/50"
                    : "text-emerald-400 border-slate-700"
                }`}
              />
              {allowParams && (
                <button
                  onClick={() => onChange(toggleParamBrackets(nodes, row.id))}
                  title="Convertir en Parámetro"
                  className={`p-1.5 rounded transition-colors font-mono text-xs font-bold tracking-tight ${String(row.value).includes("{{") ? "bg-indigo-600/20 text-indigo-400" : "bg-slate-800 text-slate-400 hover:text-white hover:bg-slate-700"}`}
                >
                  {"{{}}"}
                </button>
              )}
            </div>
          ) : (
            <div className="flex-1 flex items-center justify-end px-2">
              <button
                onClick={() => onChange(addVisualNode(nodes, row.id))}
                className="text-xs text-indigo-400 hover:text-indigo-300 flex items-center gap-1"
              >
                <Plus className="w-3 h-3" />{" "}
                {row.type === "array" ? "Agregar Ítem" : "Agregar Campo"}
              </button>
            </div>
          )}

          <button
            onClick={() => onChange(removeVisualNode(nodes, row.id))}
            className="p-1.5 text-rose-400/50 hover:text-rose-400 transition-colors"
          >
            <Trash2 className="w-4 h-4" />
          </button>
        </div>

        {(row.type === "object" || row.type === "array") && row.children && (
          <div className="w-full">
            {renderTree(row.children, level + 1, row.type)}
          </div>
        )}
      </div>
    ));
  };

  return <>{renderTree(nodes, 0, rootType)}</>;
};

// --- COMPONENTE MODAL: SCHEMA BUILDER (PESTAÑA 3) ---
const ResponseSchemaModal = ({ isOpen, onClose, apiConfig, updateConfig }) => {
  const [tab, setTab] = useState("json"); // 'json', 'visual', 'test'
  const [jsonError, setJsonError] = useState(null);

  const [testInputs, setTestInputs] = useState({});
  const [testScenario, setTestScenario] = useState("200");
  const [testLoading, setTestLoading] = useState(false);
  const [testResult, setTestResult] = useState(null);

  if (!isOpen) return null;

  const handleFormatJson = () => {
    try {
      const formatted = JSON.stringify(
        JSON.parse(apiConfig.responseSchema),
        null,
        2,
      );
      updateConfig("responseSchema", formatted);
      setJsonError(null);
    } catch (e) {
      setJsonError("No se puede formatear: JSON inválido");
    }
  };

  const handleTabSwitch = (newTab) => {
    setJsonError(null);
    if (newTab === "visual" && tab === "json") {
      try {
        const parsed = JSON.parse(apiConfig.responseSchema || "{}");
        const isArray = Array.isArray(parsed);
        updateConfig("responseSchemaRootType", isArray ? "array" : "object");
        updateConfig("responseSchemaVisual", jsonToVisualNodes(parsed));
        setTab(newTab);
      } catch (e) {
        setJsonError(
          "El JSON es inválido. Por favor, corrígelo antes de pasar al constructor visual.",
        );
      }
    } else if (newTab === "json" && tab === "visual") {
      updateConfig(
        "responseSchema",
        JSON.stringify(
          visualNodesToJson(
            apiConfig.responseSchemaVisual,
            apiConfig.responseSchemaRootType,
          ),
          null,
          2,
        ),
      );
      setTab(newTab);
    } else {
      setTab(newTab);
    }
  };

  const runMockTest = () => {
    setTestLoading(true);
    setTestResult(null);

    let simulatedUrl = apiConfig.url;
    let simulatedBody =
      apiConfig.bodyMode === "json"
        ? apiConfig.bodyRaw
        : JSON.stringify(
            visualNodesToJson(apiConfig.bodyVisual, apiConfig.bodyRootType),
            null,
            2,
          );

    Object.keys(testInputs).forEach((key) => {
      const regex = new RegExp(`\\{\\{${key}\\}\\}`, "g");
      simulatedUrl = simulatedUrl.replace(regex, testInputs[key] || "");
      simulatedBody = simulatedBody.replace(regex, testInputs[key] || "");
    });

    const simulatedHeaders = apiConfig.headers
      .filter((h) => h.key)
      .reduce((acc, h) => {
        let val = h.value;
        Object.keys(testInputs).forEach((key) => {
          val = val.replace(
            new RegExp(`\\{\\{${key}\\}\\}`, "g"),
            testInputs[key] || "",
          );
        });
        acc[h.key] = val;
        return acc;
      }, {});

    const reqMock = {
      method: apiConfig.method,
      url: simulatedUrl,
      headers: simulatedHeaders,
      body: ["POST", "PUT", "PATCH"].includes(apiConfig.method)
        ? simulatedBody
        : null,
    };

    setTimeout(() => {
      let mockRes;
      if (testScenario === "200") {
        mockRes = {
          status: 200,
          body: JSON.stringify(
            {
              search: "*",
              total_found: 1,
              total_record: 1,
              size: 1,
              index: "chile-campuses-master",
              results: [
                {
                  included: true,
                  campus_premium_pack: false,
                  serialized_at: "2026-01-16T14:01:49.292049",
                },
              ],
              domain: "example",
              total_record_included: 1,
              total_record_nonincluded: 0,
              results_nonincluded: [],
            },
            null,
            2,
          ),
        };
      } else if (testScenario === "404") {
        mockRes = {
          status: 404,
          body: JSON.stringify(
            { error: "Not Found", message: "El recurso solicitado no existe." },
            null,
            2,
          ),
        };
      } else {
        mockRes = {
          status: 500,
          body: JSON.stringify(
            { error: "Internal Server Error", code: "SRV_ERR_99" },
            null,
            2,
          ),
        };
      }
      setTestResult({ request: reqMock, response: mockRes });
      setTestLoading(false);
    }, 1200);
  };

  const useResultAsSchema = () => {
    if (testResult && testResult.response.body) {
      updateConfig("responseSchema", testResult.response.body);
      setTab("json");
    }
  };

  return (
    <div className="fixed inset-0 bg-black/70 backdrop-blur-sm z-50 flex items-center justify-center p-4 animate-in fade-in">
      <div className="bg-slate-900 border border-slate-700 rounded-xl shadow-2xl w-full max-w-5xl flex flex-col max-h-[90vh] overflow-hidden">
        {/* Modal Header */}
        <div className="px-6 py-4 border-b border-slate-800 flex justify-between items-center bg-slate-800/80">
          <h2 className="text-lg font-bold text-white flex items-center gap-2">
            <GitCommit className="w-5 h-5 text-indigo-400" /> Constructor de
            Response Schema
          </h2>
          <button onClick={onClose} className="text-slate-400 hover:text-white">
            <X className="w-5 h-5" />
          </button>
        </div>

        {/* Modal Tabs */}
        <div className="flex border-b border-slate-800 bg-slate-900 px-6 pt-4 gap-6">
          <button
            onClick={() => handleTabSwitch("json")}
            className={`pb-3 text-sm font-medium border-b-2 transition-colors ${tab === "json" ? "border-indigo-500 text-indigo-400" : "border-transparent text-slate-400 hover:text-white"}`}
          >
            <FileJson className="w-4 h-4 inline-block mr-1 mb-0.5" /> Pegar
            Ejemplo (Manual)
          </button>
          <button
            onClick={() => handleTabSwitch("visual")}
            className={`pb-3 text-sm font-medium border-b-2 transition-colors ${tab === "visual" ? "border-indigo-500 text-indigo-400" : "border-transparent text-slate-400 hover:text-white"}`}
          >
            <ListTree className="w-4 h-4 inline-block mr-1 mb-0.5" /> Armar
            Visualmente
          </button>
          <button
            onClick={() => handleTabSwitch("test")}
            className={`pb-3 text-sm font-medium border-b-2 transition-colors ${tab === "test" ? "border-indigo-500 text-indigo-400" : "border-transparent text-slate-400 hover:text-white"}`}
          >
            <Terminal className="w-4 h-4 inline-block mr-1 mb-0.5" /> Obtener de
            Prueba Real
          </button>
        </div>

        {/* Modal Content */}
        <div className="p-6 flex-1 overflow-y-auto custom-scrollbar bg-slate-900">
          {/* TAB: JSON */}
          {tab === "json" && (
            <div className="h-full flex flex-col animate-in fade-in">
              <div className="flex justify-between items-center mb-3">
                <p className="text-sm text-slate-400">
                  Pega un ejemplo del JSON de respuesta esperado.
                </p>
                {jsonError && (
                  <span className="text-xs text-rose-400 flex items-center gap-1">
                    <AlertCircle className="w-3 h-3" /> {jsonError}
                  </span>
                )}
              </div>
              <div className="relative group flex-1">
                <textarea
                  value={apiConfig.responseSchema}
                  onChange={(e) => {
                    updateConfig("responseSchema", e.target.value);
                    setJsonError(null);
                  }}
                  className="w-full h-full min-h-[400px] bg-slate-950 border border-slate-700 rounded-lg p-4 text-emerald-400 font-mono text-sm focus:border-indigo-500 outline-none resize-none custom-scrollbar"
                  placeholder={'{\n  "status": "success"\n}'}
                  spellCheck="false"
                />
                <button
                  onClick={handleFormatJson}
                  title="Formatear JSON"
                  className="absolute bottom-4 right-6 p-2 bg-slate-800 text-emerald-400 rounded-lg opacity-0 group-hover:opacity-100 transition-opacity border border-slate-700 shadow-lg hover:bg-slate-700"
                >
                  <AlignLeft className="w-5 h-5" />
                </button>
              </div>
            </div>
          )}

          {/* TAB: VISUAL */}
          {tab === "visual" && (
            <div className="h-full flex flex-col animate-in fade-in">
              <div className="flex justify-between items-center mb-4">
                <div className="flex items-center gap-2">
                  <span className="text-sm text-slate-400 font-medium">
                    Estructura Raíz:
                  </span>
                  <select
                    value={apiConfig.responseSchemaRootType}
                    onChange={(e) =>
                      updateConfig("responseSchemaRootType", e.target.value)
                    }
                    className="bg-slate-950 border border-slate-700 rounded px-2 py-1 text-xs text-slate-300 focus:outline-none"
                  >
                    <option value="object">Objeto {"{}"}</option>
                    <option value="array">Arreglo {"[]"}</option>
                  </select>
                </div>
                <button
                  onClick={() => {
                    const newNode = {
                      id: Date.now().toString(),
                      key: "",
                      type: "string",
                      value: "",
                      children: [],
                    };
                    updateConfig("responseSchemaVisual", [
                      ...apiConfig.responseSchemaVisual,
                      newNode,
                    ]);
                  }}
                  className="text-xs text-indigo-400 flex items-center gap-1 hover:text-indigo-300 bg-indigo-900/30 px-3 py-1.5 rounded"
                >
                  <Plus className="w-3 h-3" />{" "}
                  {apiConfig.responseSchemaRootType === "array"
                    ? "Agregar Ítem Raíz"
                    : "Agregar Campo Raíz"}
                </button>
              </div>
              <div className="flex-1 bg-slate-950 border border-slate-700 rounded-lg p-4 overflow-y-auto">
                {apiConfig.responseSchemaVisual.length === 0 ? (
                  <p className="text-sm text-slate-500 italic mt-4 text-center">
                    No hay campos definidos.
                  </p>
                ) : (
                  <VisualTreeEditor
                    nodes={apiConfig.responseSchemaVisual}
                    onChange={(newNodes) =>
                      updateConfig("responseSchemaVisual", newNodes)
                    }
                    rootType={apiConfig.responseSchemaRootType}
                    allowParams={false} // No se permiten {{variables}} en el schema de respuesta
                  />
                )}
              </div>
            </div>
          )}

          {/* TAB: TEST */}
          {tab === "test" && (
            <div className="grid grid-cols-12 gap-6 h-full animate-in fade-in">
              <div className="col-span-4 flex flex-col border-r border-slate-800 pr-6">
                <h3 className="text-sm font-medium text-slate-200 mb-4 border-b border-slate-700 pb-2">
                  Parámetros de Prueba
                </h3>
                <div className="flex-1 overflow-y-auto pr-2 custom-scrollbar">
                  {Object.keys(apiConfig.variables).length > 0 ? (
                    <div className="space-y-4">
                      {Object.entries(apiConfig.variables).map(
                        ([name, config]) => (
                          <div key={name}>
                            <label className="block text-xs text-slate-400 mb-1">
                              {name}{" "}
                              <span className="text-slate-600">
                                ({config.type})
                              </span>
                              {config.required && (
                                <span className="text-rose-500 ml-1">*</span>
                              )}
                            </label>
                            <input
                              type={
                                config.type === "number" ? "number" : "text"
                              }
                              value={testInputs[name] || ""}
                              onChange={(e) =>
                                setTestInputs({
                                  ...testInputs,
                                  [name]: e.target.value,
                                })
                              }
                              className="w-full bg-slate-950 border border-slate-700 rounded px-3 py-2 text-sm text-white focus:border-indigo-500"
                              placeholder={`Valor para {{${name}}}`}
                            />
                          </div>
                        ),
                      )}
                    </div>
                  ) : (
                    <p className="text-sm text-slate-500 italic">
                      La API no requiere variables dinámicas.
                    </p>
                  )}
                </div>

                <div className="mt-4 pt-4 border-t border-slate-800">
                  <label className="block text-xs text-slate-400 mb-2">
                    Simular Escenario (Mock)
                  </label>
                  <select
                    value={testScenario}
                    onChange={(e) => setTestScenario(e.target.value)}
                    className="w-full bg-slate-950 border border-slate-700 rounded px-3 py-2 text-sm text-slate-200 focus:outline-none mb-3"
                  >
                    <option value="200">Éxito (200 OK)</option>
                    <option value="404">No Encontrado (404 Not Found)</option>
                    <option value="500">
                      Error Servidor (500 Internal Error)
                    </option>
                  </select>

                  <button
                    onClick={runMockTest}
                    disabled={testLoading}
                    className="w-full py-2 bg-indigo-600 hover:bg-indigo-700 text-white font-medium rounded-lg flex justify-center items-center gap-2 transition-colors disabled:opacity-50"
                  >
                    {testLoading ? (
                      "Llamando..."
                    ) : (
                      <>
                        <Play className="w-4 h-4" /> Ejecutar Petición
                      </>
                    )}
                  </button>
                </div>
              </div>

              <div className="col-span-8 flex flex-col h-full">
                <div className="flex justify-between items-center mb-2">
                  <span className="text-sm font-medium text-slate-200">
                    Inspección de Tráfico
                  </span>
                  {testResult && testResult.response.status < 300 && (
                    <button
                      onClick={useResultAsSchema}
                      className="text-xs bg-emerald-600/20 text-emerald-400 border border-emerald-500/30 px-3 py-1.5 rounded hover:bg-emerald-600/40 transition-colors flex items-center gap-1"
                    >
                      <Check className="w-3.5 h-3.5" /> Usar como Schema
                    </button>
                  )}
                </div>

                <div className="flex-1 bg-black/50 border border-slate-800 rounded-lg overflow-hidden flex flex-col">
                  {!testResult ? (
                    <div className="flex-1 flex items-center justify-center text-slate-600 text-sm italic">
                      Esperando ejecución de la petición...
                    </div>
                  ) : (
                    <div className="flex flex-col h-full">
                      <div className="flex-1 overflow-y-auto custom-scrollbar flex flex-col min-h-0 border-b-4 border-slate-900">
                        <div className="bg-slate-900/80 px-4 py-2 flex justify-between sticky top-0">
                          <span className="text-[10px] font-bold text-slate-400 uppercase tracking-widest">
                            REQUEST ENVIADO
                          </span>
                        </div>
                        <pre className="p-4 text-xs font-mono text-slate-300 whitespace-pre-wrap">
                          <div className="mb-2">
                            <span className="text-indigo-400 font-bold">
                              {testResult.request.method}
                            </span>{" "}
                            {testResult.request.url}
                          </div>
                          {Object.entries(testResult.request.headers).length >
                            0 && (
                            <div className="mb-2 text-slate-400">
                              {Object.entries(testResult.request.headers).map(
                                ([k, v]) => (
                                  <div key={k}>
                                    {k}: {v}
                                  </div>
                                ),
                              )}
                            </div>
                          )}
                          {testResult.request.body && (
                            <div className="text-emerald-500/70 mt-3 pt-3 border-t border-slate-800/50">
                              {testResult.request.body}
                            </div>
                          )}
                        </pre>
                      </div>

                      <div className="flex-1 overflow-y-auto custom-scrollbar flex flex-col min-h-0">
                        <div className="bg-slate-900/80 px-4 py-2 flex justify-between items-center sticky top-0">
                          <span className="text-[10px] font-bold text-slate-400 uppercase tracking-widest">
                            RESPUESTA OBTENIDA
                          </span>
                          <span
                            className={`text-[10px] font-bold px-2 py-0.5 rounded ${testResult.response.status < 300 ? "bg-emerald-950/50 text-emerald-400" : "bg-rose-950/50 text-rose-400"}`}
                          >
                            STATUS {testResult.response.status}
                          </span>
                        </div>
                        <pre
                          className={`p-4 text-xs font-mono whitespace-pre-wrap flex-1 ${testResult.response.status < 300 ? "text-emerald-400" : "text-rose-400"}`}
                        >
                          {testResult.response.body}
                        </pre>
                      </div>
                    </div>
                  )}
                </div>
              </div>
            </div>
          )}
        </div>

        <div className="px-6 py-4 border-t border-slate-800 bg-slate-800/30 flex justify-end">
          <button
            onClick={() => {
              handleTabSwitch("json");
              onClose();
            }}
            className="px-5 py-2 bg-indigo-600 hover:bg-indigo-700 text-white rounded-lg text-sm font-medium transition-colors"
          >
            Confirmar Schema
          </button>
        </div>
      </div>
    </div>
  );
};

// --- COMPONENTE PRINCIPAL (WIZARD) ---

export default function App() {
  const [activeTab, setActiveTab] = useState("url");
  const [saveStatus, setSaveStatus] = useState(null);
  const [isVariablesModalOpen, setIsVariablesModalOpen] = useState(false);
  const [isResponseModalOpen, setIsResponseModalOpen] = useState(false);
  const [jsonError, setJsonError] = useState(null);

  const steps = ["url", "request", "validation", "test"];
  const currentStepIndex = steps.indexOf(activeTab);

  const initialJsonStr = `{
  "fields_required": [
    "institution_name",
    "institution_code"
  ],
  "filter_match": [
    {
      "fieldname": "campus_code",
      "fieldvalues": [
        "{{campus_code}}"
      ]
    }
  ],
  "search_size": 1
}`;

  const initialResponseExample = `{
  "search": "*",
  "total_found": 1,
  "total_record": 1,
  "size": 1,
  "index": "chile-campuses-master",
  "results": [
    {
      "included": true,
      "campus_premium_pack": false,
      "serialized_at": "2026-01-16T14:01:49.292049"
    }
  ],
  "domain": "example",
  "total_record_included": 1,
  "total_record_nonincluded": 0,
  "results_nonincluded": []
}`;

  const [apiConfig, setApiConfig] = useState({
    name: "Búsqueda de Institución",
    method: "POST",
    url: "https://api.{{env}}.midominio.com/v2/search",
    headers: [
      { id: 1, key: "Authorization", value: "Bearer static_token_123" },
    ],
    bodyMode: "json",
    bodyRootType: "object",
    bodyRaw: initialJsonStr,
    bodyVisual: jsonToVisualNodes(JSON.parse(initialJsonStr)),
    validationMode: "both",
    httpCodeMode: "2xx",
    httpCodeCustom: "",
    responseSchema: initialResponseExample,
    responseSchemaVisual: jsonToVisualNodes(JSON.parse(initialResponseExample)),
    responseSchemaRootType: "object",
    expression: "response.results.length > 0",
    variables: {},
  });

  const availableTokens = useMemo(() => {
    const paths = new Set();
    paths.add("response");

    try {
      const obj = JSON.parse(apiConfig.responseSchema);
      const traverse = (o, currentPath) => {
        if (o !== null && typeof o === "object") {
          if (Array.isArray(o)) {
            if (o.length > 0) traverse(o[0], currentPath);
          } else {
            for (const key in o) {
              const newPath = `${currentPath}.${key}`;
              paths.add(newPath);
              traverse(o[key], newPath);
            }
          }
        }
      };
      traverse(obj, "response");
    } catch {}

    Object.keys(apiConfig.variables).forEach((varName) => {
      paths.add(`{{${varName}}}`);
    });

    return Array.from(paths);
  }, [apiConfig.responseSchema, apiConfig.variables]);

  useEffect(() => {
    const extractVarsFromText = (text, location) => {
      const regex = /\{\{([^}]+)\}\}/g;
      let match;
      const extracted = [];
      while ((match = regex.exec(text)) !== null) {
        extracted.push({ name: match[1], location });
      }
      return extracted;
    };

    let allExtracted = [];

    const qIndex = apiConfig.url.indexOf("?");
    const protoIndex = apiConfig.url.indexOf("://");
    const pathIndex = apiConfig.url.indexOf(
      "/",
      protoIndex !== -1 ? protoIndex + 3 : 0,
    );

    const urlRegex = /\{\{([^}]+)\}\}/g;
    let match;
    while ((match = urlRegex.exec(apiConfig.url)) !== null) {
      const name = match[1];
      const index = match.index;
      let loc = "url_domain";
      if (qIndex !== -1 && index > qIndex) loc = "url_query";
      else if (pathIndex !== -1 && index > pathIndex) loc = "url_path";
      allExtracted.push({ name, location: loc });
    }

    apiConfig.headers.forEach((h) => {
      allExtracted.push(...extractVarsFromText(h.key, "header_key"));
      allExtracted.push(...extractVarsFromText(h.value, "header_value"));
    });

    if (["POST", "PUT", "PATCH"].includes(apiConfig.method)) {
      if (apiConfig.bodyMode === "json") {
        allExtracted.push(...extractVarsFromText(apiConfig.bodyRaw, "body"));
      } else {
        const extractFromVisual = (nodes) => {
          nodes.forEach((n) => {
            allExtracted.push(...extractVarsFromText(String(n.value), "body"));
            if (n.children && n.children.length > 0)
              extractFromVisual(n.children);
          });
        };
        extractFromVisual(apiConfig.bodyVisual);
      }
    }

    setApiConfig((prev) => {
      const newVars = { ...prev.variables };
      const currentDetectedNames = new Set(allExtracted.map((v) => v.name));

      Object.keys(newVars).forEach((key) => {
        if (!currentDetectedNames.has(key)) delete newVars[key];
      });

      allExtracted.forEach(({ name, location }) => {
        if (!newVars[name]) {
          const isUrlReq =
            location.startsWith("url_domain") ||
            location.startsWith("url_path");
          newVars[name] = {
            type: "any",
            required: isUrlReq,
            locations: new Set([location]),
          };
        } else {
          newVars[name].locations.add(location);
        }
      });

      return { ...prev, variables: newVars };
    });
  }, [
    apiConfig.url,
    apiConfig.headers,
    apiConfig.bodyRaw,
    apiConfig.bodyVisual,
    apiConfig.bodyMode,
    apiConfig.method,
  ]);

  const updateConfig = (key, value) =>
    setApiConfig((prev) => ({ ...prev, [key]: value }));

  const handleFormatJson = (targetField) => {
    try {
      const formatted = JSON.stringify(
        JSON.parse(apiConfig[targetField]),
        null,
        2,
      );
      updateConfig(targetField, formatted);
      if (targetField === "bodyRaw") setJsonError(null);
    } catch (e) {
      if (targetField === "bodyRaw")
        setJsonError("No se puede formatear: JSON inválido");
    }
  };

  const handleBodyModeSwitch = (newMode) => {
    setJsonError(null);
    if (newMode === "visual" && apiConfig.bodyMode === "json") {
      try {
        const parsed = JSON.parse(apiConfig.bodyRaw || "{}");
        const isArray = Array.isArray(parsed);
        updateConfig("bodyRootType", isArray ? "array" : "object");
        updateConfig("bodyVisual", jsonToVisualNodes(parsed));
        updateConfig("bodyMode", newMode);
      } catch (e) {
        setJsonError(
          "El JSON es inválido. Por favor, corrígelo antes de pasar al constructor visual.",
        );
      }
    } else if (newMode === "json" && apiConfig.bodyMode === "visual") {
      updateConfig(
        "bodyRaw",
        JSON.stringify(
          visualNodesToJson(apiConfig.bodyVisual, apiConfig.bodyRootType),
          null,
          2,
        ),
      );
      updateConfig("bodyMode", newMode);
    } else {
      updateConfig("bodyMode", newMode);
    }
  };

  const updateVariableConfig = (name, field, value) => {
    setApiConfig((prev) => ({
      ...prev,
      variables: {
        ...prev.variables,
        [name]: { ...prev.variables[name], [field]: value },
      },
    }));
  };

  const handleNext = () =>
    currentStepIndex < steps.length - 1 &&
    setActiveTab(steps[currentStepIndex + 1]);
  const handlePrev = () =>
    currentStepIndex > 0 && setActiveTab(steps[currentStepIndex - 1]);
  const handleSave = () => {
    console.log(apiConfig);
    setSaveStatus("success");
    setTimeout(() => setSaveStatus(null), 3000);
  };

  const renderTabs = () => (
    <div className="flex justify-between items-center mb-6">
      <div className="flex space-x-1 bg-slate-800 p-1 rounded-lg w-max">
        {[
          { id: "url", icon: Globe, label: "1. Ruta & URL" },
          { id: "request", icon: Braces, label: "2. Headers & Body" },
          { id: "validation", icon: Check, label: "3. Validación" },
          { id: "test", icon: Play, label: "4. Probar" },
        ].map((tab) => (
          <button
            key={tab.id}
            onClick={() => setActiveTab(tab.id)}
            className={`flex items-center space-x-2 px-4 py-2 rounded-md text-sm font-medium transition-colors ${
              activeTab === tab.id
                ? "bg-indigo-600 text-white"
                : "text-slate-400 hover:text-white hover:bg-slate-700"
            }`}
          >
            <tab.icon className="w-4 h-4" />
            <span>{tab.label}</span>
          </button>
        ))}
      </div>

      <button
        onClick={() => setIsVariablesModalOpen(true)}
        className="flex items-center gap-2 bg-indigo-900/30 text-indigo-300 border border-indigo-500/30 px-4 py-2 rounded-lg text-sm font-medium hover:bg-indigo-900/50 transition-colors shadow-lg shadow-indigo-900/20"
      >
        <ListTree className="w-4 h-4" />
        Gestor de Variables ({Object.keys(apiConfig.variables).length})
      </button>
    </div>
  );

  return (
    <div className="min-h-screen bg-slate-900 text-slate-200 p-8 font-sans">
      <div className="max-w-6xl mx-auto">
        <div className="mb-8">
          <h1 className="text-2xl font-bold text-white flex items-center gap-2">
            <Settings className="w-6 h-6 text-indigo-400" /> Constructor de API
            Externa
          </h1>
          <p className="text-slate-400 text-sm mt-1">
            Configura un endpoint externo con valores dinámicos para ser
            evaluado en el motor de reglas.
          </p>
        </div>

        {renderTabs()}

        <div className="bg-slate-800 border border-slate-700 rounded-xl p-6 shadow-xl min-h-[500px] flex flex-col">
          {/* TAB 1: URL Y PARAMS */}
          {activeTab === "url" && (
            <div className="space-y-6 animate-in fade-in slide-in-from-bottom-2 flex-1">
              <div>
                <label className="block text-sm font-medium text-slate-400 mb-1">
                  Nombre de la Integración
                </label>
                <input
                  type="text"
                  value={apiConfig.name}
                  onChange={(e) => updateConfig("name", e.target.value)}
                  className="w-full md:w-2/3 bg-slate-900 border border-slate-700 rounded-lg px-4 py-2 text-white focus:outline-none focus:border-indigo-500"
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-slate-400 mb-1">
                  Configuración de URL
                </label>
                <div className="flex gap-2">
                  <select
                    value={apiConfig.method}
                    onChange={(e) => updateConfig("method", e.target.value)}
                    className="bg-slate-900 border border-slate-700 rounded-lg px-4 py-2 text-white font-mono focus:outline-none focus:border-indigo-500 w-32"
                  >
                    {["GET", "POST", "PUT", "PATCH", "DELETE"].map((m) => (
                      <option key={m} value={m}>
                        {m}
                      </option>
                    ))}
                  </select>
                  <HighlightedInput
                    value={apiConfig.url}
                    onChange={(val) => updateConfig("url", val)}
                    placeholder="https://api.{{env}}.midominio.com/path/{{id}}?query={{valor}}"
                    className="flex-1 bg-slate-900 border border-slate-700 rounded-lg focus-within:border-indigo-500 transition-colors"
                  />
                </div>
                <p className="text-xs text-slate-500 mt-2 flex items-center gap-1">
                  <AlertCircle className="w-3 h-3" /> Usa doble llave{" "}
                  <code>{`{{parametro}}`}</code> para definir variables
                  dinámicas.
                </p>
              </div>
            </div>
          )}

          {/* TAB 2: HEADERS & BODY */}
          {activeTab === "request" && (
            <div className="space-y-8 animate-in fade-in slide-in-from-bottom-2 flex-1">
              <section>
                <div className="flex justify-between items-center mb-4 border-b border-slate-700 pb-2">
                  <h3 className="text-base font-medium text-slate-200">
                    Headers
                  </h3>
                  <button
                    onClick={() =>
                      updateConfig("headers", [
                        ...apiConfig.headers,
                        { id: Date.now(), key: "", value: "" },
                      ])
                    }
                    className="text-xs flex items-center gap-1 text-indigo-400 hover:text-indigo-300 transition-colors"
                  >
                    <Plus className="w-4 h-4" /> Agregar Header
                  </button>
                </div>
                {apiConfig.headers.length === 0 ? (
                  <p className="text-sm text-slate-500 italic">
                    No hay headers configurados.
                  </p>
                ) : (
                  <div className="space-y-3">
                    {apiConfig.headers.map((header) => (
                      <div key={header.id} className="flex gap-3 items-start">
                        <HighlightedInput
                          value={header.key}
                          onChange={(val) =>
                            updateConfig(
                              "headers",
                              apiConfig.headers.map((h) =>
                                h.id === header.id ? { ...h, key: val } : h,
                              ),
                            )
                          }
                          placeholder="Key (ej. Authorization)"
                          className="w-1/3 bg-slate-900 border border-slate-700 rounded-lg focus-within:border-indigo-500 h-10"
                        />
                        <HighlightedInput
                          value={header.value}
                          onChange={(val) =>
                            updateConfig(
                              "headers",
                              apiConfig.headers.map((h) =>
                                h.id === header.id ? { ...h, value: val } : h,
                              ),
                            )
                          }
                          placeholder="Valor o {{parametro}}"
                          className="flex-1 bg-slate-900 border border-slate-700 rounded-lg focus-within:border-indigo-500 h-10"
                        />
                        <button
                          onClick={() =>
                            updateConfig(
                              "headers",
                              apiConfig.headers.filter(
                                (h) => h.id !== header.id,
                              ),
                            )
                          }
                          className="h-10 px-2 text-rose-400 hover:text-rose-300"
                        >
                          <Trash2 className="w-4 h-4" />
                        </button>
                      </div>
                    ))}
                  </div>
                )}
              </section>

              {["POST", "PUT", "PATCH"].includes(apiConfig.method) && (
                <section>
                  <div className="flex justify-between items-center mb-4 border-b border-slate-700 pb-2">
                    <h3 className="text-base font-medium text-slate-200">
                      Request Body
                    </h3>
                    <div className="flex bg-slate-900 rounded-lg p-1 items-center gap-4">
                      {jsonError && (
                        <span className="text-xs text-rose-400 flex items-center gap-1">
                          <AlertCircle className="w-3 h-3" /> {jsonError}
                        </span>
                      )}

                      <div className="flex gap-1">
                        <button
                          onClick={() => handleBodyModeSwitch("visual")}
                          className={`px-3 py-1 text-xs font-medium rounded-md ${apiConfig.bodyMode === "visual" ? "bg-slate-700 text-white" : "text-slate-400 hover:text-white"}`}
                        >
                          Constructor Visual
                        </button>
                        <button
                          onClick={() => handleBodyModeSwitch("json")}
                          className={`px-3 py-1 text-xs font-medium rounded-md ${apiConfig.bodyMode === "json" ? "bg-slate-700 text-white" : "text-slate-400 hover:text-white"}`}
                        >
                          JSON Crudo
                        </button>
                      </div>
                    </div>
                  </div>

                  {apiConfig.bodyMode === "json" ? (
                    <div className="relative group">
                      <p className="text-xs text-slate-500 mb-2">
                        Escribe tu JSON. Usa <code>{`{{parametro}}`}</code> para
                        inyectar variables.
                      </p>
                      <HighlightedTextarea
                        value={apiConfig.bodyRaw}
                        onChange={(val) => {
                          updateConfig("bodyRaw", val);
                          setJsonError(null);
                        }}
                        placeholder={'{\n  "key": "value"\n}'}
                        className="w-full h-[400px] bg-slate-900 border border-slate-700 rounded-lg focus-within:border-indigo-500"
                      />
                      <button
                        onClick={() => handleFormatJson("bodyRaw")}
                        title="Formatear JSON"
                        className="absolute bottom-4 right-6 p-2 bg-slate-800/80 text-emerald-400 rounded-lg opacity-0 group-hover:opacity-100 transition-opacity border border-slate-700 shadow-lg hover:bg-slate-700"
                      >
                        <AlignLeft className="w-5 h-5" />
                      </button>
                    </div>
                  ) : (
                    <div className="grid grid-cols-12 gap-6">
                      <div className="col-span-12 lg:col-span-7">
                        <div className="flex justify-between items-center mb-2">
                          <div className="flex items-center gap-2">
                            <span className="text-xs text-slate-400 font-medium uppercase">
                              Estructura del Body
                            </span>
                            <select
                              value={apiConfig.bodyRootType}
                              onChange={(e) =>
                                updateConfig("bodyRootType", e.target.value)
                              }
                              className="bg-slate-900 border border-slate-700 rounded px-2 py-1 text-[10px] text-slate-300 focus:outline-none"
                            >
                              <option value="object">Objeto {"{}"}</option>
                              <option value="array">Arreglo {"[]"}</option>
                            </select>
                          </div>
                          <button
                            onClick={() => {
                              const newNode = {
                                id: Date.now().toString(),
                                key: "",
                                type: "string",
                                value: "",
                                children: [],
                              };
                              updateConfig("bodyVisual", [
                                ...apiConfig.bodyVisual,
                                newNode,
                              ]);
                            }}
                            className="text-xs text-indigo-400 flex items-center gap-1 hover:text-indigo-300"
                          >
                            <Plus className="w-3 h-3" />{" "}
                            {apiConfig.bodyRootType === "array"
                              ? "Agregar Ítem Raíz"
                              : "Agregar Campo Raíz"}
                          </button>
                        </div>
                        <div className="max-h-[400px] overflow-y-auto pr-2 custom-scrollbar pb-4">
                          {apiConfig.bodyVisual.length === 0 ? (
                            <p className="text-sm text-slate-500 italic mt-4 text-center">
                              No hay campos definidos.
                            </p>
                          ) : (
                            <VisualTreeEditor
                              nodes={apiConfig.bodyVisual}
                              onChange={(newNodes) =>
                                updateConfig("bodyVisual", newNodes)
                              }
                              rootType={apiConfig.bodyRootType}
                            />
                          )}
                        </div>
                      </div>

                      <div className="col-span-12 lg:col-span-5 bg-slate-900 rounded-lg p-4 border border-slate-700 flex flex-col h-[400px]">
                        <span className="text-xs text-slate-400 font-medium uppercase mb-2">
                          Preview (Solo Lectura)
                        </span>
                        <pre className="flex-1 text-xs font-mono overflow-auto custom-scrollbar">
                          {(() => {
                            const jsonString = JSON.stringify(
                              visualNodesToJson(
                                apiConfig.bodyVisual,
                                apiConfig.bodyRootType,
                              ),
                              null,
                              2,
                            );
                            return renderHighlightedText(
                              jsonString,
                              "text-emerald-400",
                            );
                          })()}
                        </pre>
                      </div>
                    </div>
                  )}
                </section>
              )}
            </div>
          )}

          {/* TAB 3: VALIDATION */}
          {activeTab === "validation" && (
            <div className="space-y-6 animate-in fade-in slide-in-from-bottom-2 flex-1">
              <div className="bg-indigo-900/20 border border-indigo-500/30 rounded-lg p-4 mb-4">
                <p className="text-sm text-indigo-200">
                  Traduce la respuesta de la API a un simple{" "}
                  <strong>Verdadero</strong> o <strong>Falso</strong> para que
                  el motor de reglas tome la decisión final.
                </p>
              </div>

              {/* SECCIÓN HTTP CODE */}
              <div className="bg-slate-900/50 p-4 border border-slate-700 rounded-xl">
                <label className="block text-sm font-medium text-slate-300 mb-2">
                  1. Validación de Status HTTP
                </label>
                <div className="flex items-center gap-4">
                  <select
                    value={apiConfig.httpCodeMode}
                    onChange={(e) =>
                      updateConfig("httpCodeMode", e.target.value)
                    }
                    className="w-1/3 bg-slate-900 border border-slate-600 rounded-lg px-4 py-2.5 text-slate-200 text-sm focus:outline-none focus:border-indigo-500"
                  >
                    <option value="2xx">Cualquier Éxito (2xx)</option>
                    <option value="custom">Códigos Específicos...</option>
                  </select>

                  {apiConfig.httpCodeMode === "custom" && (
                    <input
                      type="text"
                      value={apiConfig.httpCodeCustom}
                      onChange={(e) =>
                        updateConfig("httpCodeCustom", e.target.value)
                      }
                      className="flex-1 bg-slate-900 border border-slate-600 rounded-lg px-4 py-2.5 text-white font-mono text-sm focus:border-indigo-500"
                      placeholder="Ej: 200, 201, 204"
                    />
                  )}
                </div>
                {apiConfig.httpCodeMode === "2xx" && (
                  <p className="text-xs text-slate-500 mt-2 ml-1">
                    Cualquier respuesta en el rango 200-299 será considerada
                    como un inicio exitoso.
                  </p>
                )}
              </div>

              {/* SECCIÓN BODY & EXPRESIÓN */}
              <div className="bg-slate-900/50 p-4 border border-slate-700 rounded-xl mt-6">
                <div className="flex justify-between items-center mb-4">
                  <label className="block text-sm font-medium text-slate-300">
                    2. Validación Lógica de Contenido
                  </label>
                </div>

                <div className="grid grid-cols-12 gap-6">
                  {/* Panel de Configuración de Schema (Lógica Condicional) */}
                  {(() => {
                    const hasSchema =
                      availableTokens.length >
                      Object.keys(apiConfig.variables).length + 1;

                    return hasSchema ? (
                      <div className="col-span-12 lg:col-span-4 flex flex-col bg-slate-950 border border-slate-800 rounded-lg relative overflow-hidden group h-[180px]">
                        <div className="bg-slate-900/90 backdrop-blur-sm px-3 py-2 flex justify-between items-center border-b border-slate-800 absolute top-0 w-full z-10 shadow-sm">
                          <span className="text-[10px] font-bold text-slate-400 uppercase tracking-widest flex items-center gap-1.5">
                            <FileJson className="w-3 h-3" /> Schema Definido
                          </span>
                          <button
                            onClick={() => setIsResponseModalOpen(true)}
                            className="text-indigo-400 hover:text-indigo-300 p-1.5 bg-indigo-500/10 hover:bg-indigo-500/20 rounded transition-colors"
                            title="Modificar Schema"
                          >
                            <Edit3 className="w-3.5 h-3.5" />
                          </button>
                        </div>
                        <pre className="p-4 pt-12 text-[10px] font-mono text-slate-400 overflow-y-auto custom-scrollbar flex-1 whitespace-pre-wrap">
                          {apiConfig.responseSchema}
                        </pre>
                      </div>
                    ) : (
                      <div className="col-span-12 lg:col-span-4 flex flex-col bg-slate-950 border border-slate-800 border-dashed rounded-lg p-4 justify-center items-center text-center h-[180px]">
                        <FileJson className="w-10 h-10 text-slate-600 mb-3" />
                        <h4 className="text-sm font-medium text-slate-300 mb-1">
                          Estructura de Respuesta
                        </h4>
                        <p className="text-[11px] text-slate-500 mb-4 px-2">
                          No hay un schema de respuesta configurado.
                        </p>
                        <button
                          onClick={() => setIsResponseModalOpen(true)}
                          className="px-4 py-2 bg-slate-800 hover:bg-slate-700 border border-slate-600 text-slate-200 rounded-lg text-xs font-medium transition-colors"
                        >
                          Configurar Schema
                        </button>
                      </div>
                    );
                  })()}

                  {/* Editor de Expresión */}
                  <div className="col-span-12 lg:col-span-8 flex flex-col h-[180px]">
                    <label className="text-xs font-semibold text-slate-400 uppercase tracking-wide mb-2 flex items-center gap-1.5">
                      <Code className="w-3.5 h-3.5 text-indigo-400" /> Expresión
                      a Evaluar
                    </label>
                    <ExpressionEditor
                      value={apiConfig.expression}
                      onChange={(val) => updateConfig("expression", val)}
                      tokens={availableTokens}
                    />
                  </div>
                </div>
              </div>
            </div>
          )}

          {/* TAB 4: TEST */}
          {activeTab === "test" && (
            <div className="space-y-6 animate-in fade-in slide-in-from-bottom-2 flex-1">
              <div className="grid grid-cols-2 gap-8 h-full">
                <div>
                  <h3 className="text-base font-medium text-slate-200 mb-4 border-b border-slate-700 pb-2">
                    Valores para la Prueba
                  </h3>

                  {Object.keys(apiConfig.variables).length > 0 ? (
                    <div className="space-y-3 mb-6">
                      {Object.entries(apiConfig.variables).map(
                        ([name, config]) => (
                          <div key={name}>
                            <label className="block text-xs text-slate-400 mb-1">
                              {name}{" "}
                              <span className="text-slate-600">
                                ({config.type})
                              </span>
                              {config.required && (
                                <span className="text-rose-500 ml-1">*</span>
                              )}
                            </label>
                            <input
                              type={
                                config.type === "number" ? "number" : "text"
                              }
                              className="w-full bg-slate-900 border border-slate-700 rounded px-3 py-1.5 text-sm text-white focus:border-indigo-500"
                              placeholder={`Valor para {{${name}}}`}
                            />
                          </div>
                        ),
                      )}
                    </div>
                  ) : (
                    <p className="text-sm text-slate-500 italic">
                      No hay variables dinámicas configuradas.
                    </p>
                  )}

                  <button className="w-full py-2 mt-4 bg-indigo-600 hover:bg-indigo-700 text-white font-medium rounded-lg flex justify-center items-center gap-2 transition-colors">
                    <Play className="w-4 h-4" /> Ejecutar Evaluación Final
                  </button>
                </div>

                <div className="bg-black/40 rounded-xl border border-slate-800 flex flex-col overflow-hidden h-96">
                  <div className="bg-slate-800/80 px-4 py-2 border-b border-slate-700 flex justify-between items-center">
                    <span className="text-xs font-medium text-slate-300">
                      Consola de Regla
                    </span>
                  </div>
                  <div className="p-4 flex-1 overflow-y-auto font-mono text-xs space-y-4 text-slate-500">
                    Presiona "Ejecutar" para comprobar el flujo: Petición {">"}{" "}
                    Status {">"} Expresión.
                  </div>
                </div>
              </div>
            </div>
          )}

          {/* NAVEGACIÓN INFERIOR */}
          <div className="mt-auto pt-6 border-t border-slate-800 flex justify-between items-center">
            <div>
              {currentStepIndex > 0 && (
                <button
                  onClick={handlePrev}
                  className="px-5 py-2 text-sm font-medium text-slate-300 hover:text-white hover:bg-slate-800 rounded-lg flex items-center gap-2"
                >
                  <ArrowLeft className="w-4 h-4" /> Atrás
                </button>
              )}
            </div>
            <div>
              {currentStepIndex < steps.length - 1 ? (
                <button
                  onClick={handleNext}
                  className="px-5 py-2 text-sm font-medium bg-indigo-600 hover:bg-indigo-700 text-white rounded-lg flex items-center gap-2"
                >
                  Siguiente <ArrowRight className="w-4 h-4" />
                </button>
              ) : (
                <button
                  onClick={handleSave}
                  disabled={saveStatus === "success"}
                  className={`px-5 py-2 text-sm font-medium text-white rounded-lg flex items-center gap-2 ${saveStatus === "success" ? "bg-emerald-600" : "bg-indigo-600 hover:bg-indigo-700"}`}
                >
                  {saveStatus === "success" ? (
                    <>
                      ¡Guardada! <Check className="w-4 h-4" />
                    </>
                  ) : (
                    <>
                      Finalizar <Check className="w-4 h-4" />
                    </>
                  )}
                </button>
              )}
            </div>
          </div>
        </div>
      </div>

      {/* MODAL GESTOR DE VARIABLES */}
      {isVariablesModalOpen && (
        <div className="fixed inset-0 bg-black/60 backdrop-blur-sm z-50 flex items-center justify-center p-4 animate-in fade-in">
          <div className="bg-slate-900 border border-slate-700 rounded-xl shadow-2xl w-full max-w-3xl overflow-hidden flex flex-col max-h-[80vh]">
            <div className="px-6 py-4 border-b border-slate-800 flex justify-between items-center bg-slate-800/50">
              <h2 className="text-lg font-bold text-white flex items-center gap-2">
                <ListTree className="w-5 h-5 text-indigo-400" /> Variables
                Dinámicas del Sistema
              </h2>
              <button
                onClick={() => setIsVariablesModalOpen(false)}
                className="text-slate-400 hover:text-white"
              >
                <X className="w-5 h-5" />
              </button>
            </div>

            <div className="p-6 overflow-y-auto flex-1 custom-scrollbar">
              <p className="text-sm text-slate-400 mb-6">
                Aquí se listan automáticamente todas las variables (texto entre
                dobles llaves <code>{`{{}}`}</code>) que has definido en la URL,
                Headers y el Body. Configura sus propiedades para la validación
                del sistema.
              </p>

              {Object.keys(apiConfig.variables).length === 0 ? (
                <div className="text-center py-10 bg-slate-800/30 rounded-lg border border-slate-800 border-dashed">
                  <p className="text-slate-500">
                    No se han detectado variables dinámicas aún.
                  </p>
                </div>
              ) : (
                <table className="w-full text-left text-sm">
                  <thead>
                    <tr className="text-slate-400 border-b border-slate-700">
                      <th className="pb-2 font-medium">Nombre (Snake Case)</th>
                      <th className="pb-2 font-medium">Ubicaciones</th>
                      <th className="pb-2 font-medium">Tipo</th>
                      <th className="pb-2 font-medium text-center">
                        Requerido
                      </th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-slate-800/50">
                    {Object.entries(apiConfig.variables).map(
                      ([name, config]) => {
                        const isSnakeCase = /^[a-z0-9_]+$/.test(name);
                        const isUrlDomainPath = Array.from(
                          config.locations,
                        ).some((l) => l === "url_domain" || l === "url_path");

                        return (
                          <tr key={name} className="group">
                            <td className="py-3 font-mono">
                              <span
                                className={`flex items-center gap-1.5 ${isSnakeCase ? "text-indigo-300" : "text-rose-400"}`}
                              >
                                {"{{"}
                                {name}
                                {"}}"}
                                {!isSnakeCase && (
                                  <AlertCircle
                                    className="w-3.5 h-3.5"
                                    title="Error: Debe ser snake_case"
                                  />
                                )}
                              </span>
                            </td>
                            <td className="py-3">
                              <div className="flex flex-wrap gap-1">
                                {Array.from(config.locations).map((loc) => (
                                  <span
                                    key={loc}
                                    className="text-[10px] uppercase tracking-wider px-1.5 py-0.5 rounded bg-slate-800 text-slate-400 border border-slate-700"
                                  >
                                    {loc.replace("_", " ")}
                                  </span>
                                ))}
                              </div>
                            </td>
                            <td className="py-3 pr-4">
                              <select
                                value={config.type}
                                onChange={(e) =>
                                  updateVariableConfig(
                                    name,
                                    "type",
                                    e.target.value,
                                  )
                                }
                                className="w-full bg-slate-950 border border-slate-700 rounded px-2 py-1 text-slate-300 text-xs focus:outline-none focus:border-indigo-500"
                              >
                                <option value="any">Any</option>
                                <option value="string">String</option>
                                <option value="number">Number</option>
                                <option value="boolean">Boolean</option>
                              </select>
                            </td>
                            <td className="py-3 text-center">
                              <label className="inline-flex items-center cursor-pointer">
                                <input
                                  type="checkbox"
                                  checked={config.required}
                                  onChange={(e) =>
                                    updateVariableConfig(
                                      name,
                                      "required",
                                      e.target.checked,
                                    )
                                  }
                                  disabled={isUrlDomainPath}
                                  className="rounded border-slate-600 bg-slate-900 text-indigo-500 focus:ring-indigo-500 disabled:opacity-50"
                                />
                              </label>
                              {isUrlDomainPath && (
                                <div className="text-[10px] text-slate-500 mt-1">
                                  Forzado por URL
                                </div>
                              )}
                            </td>
                          </tr>
                        );
                      },
                    )}
                  </tbody>
                </table>
              )}
            </div>
            <div className="px-6 py-4 border-t border-slate-800 bg-slate-800/30 flex justify-end">
              <button
                onClick={() => setIsVariablesModalOpen(false)}
                className="px-4 py-2 bg-indigo-600 hover:bg-indigo-700 text-white rounded-lg text-sm font-medium transition-colors"
              >
                Hecho
              </button>
            </div>
          </div>
        </div>
      )}

      {/* MODAL CONFIGURADOR DE SCHEMA DE RESPUESTA */}
      <ResponseSchemaModal
        isOpen={isResponseModalOpen}
        onClose={() => setIsResponseModalOpen(false)}
        apiConfig={apiConfig}
        updateConfig={updateConfig}
      />
    </div>
  );
}
