"use client";

import { useState, type FormEvent } from "react";
import type { Question } from "@/lib/types";

const bloomColors: Record<string, string> = {
  remember: "bg-purple-100 text-purple-800",
  understand: "bg-blue-100 text-blue-800",
  apply: "bg-green-100 text-green-800",
  analyze: "bg-yellow-100 text-yellow-800",
  evaluate: "bg-orange-100 text-orange-800",
  create: "bg-red-100 text-red-800",
};

const difficultyColors: Record<string, string> = {
  easy: "bg-green-100 text-green-700",
  medium: "bg-yellow-100 text-yellow-700",
  hard: "bg-red-100 text-red-700",
};

interface QuestionCardProps {
  question: Question;
  onAnswer: (answer: string) => void;
  disabled?: boolean;
}

export default function QuestionCard({
  question,
  onAnswer,
  disabled = false,
}: QuestionCardProps) {
  const [selected, setSelected] = useState("");

  function handleSubmit(e: FormEvent) {
    e.preventDefault();
    if (selected.trim()) {
      onAnswer(selected.trim());
    }
  }

  const bloomBadge =
    bloomColors[question.bloomLevel] ?? "bg-gray-100 text-gray-800";
  const diffBadge =
    difficultyColors[question.difficulty] ?? "bg-gray-100 text-gray-800";

  return (
    <form
      onSubmit={handleSubmit}
      className="rounded-lg border border-gray-200 bg-white p-6 shadow-sm"
    >
      {/* Tags row */}
      <div className="mb-4 flex flex-wrap gap-2">
        <span
          className={`rounded-full px-2.5 py-0.5 text-xs font-medium capitalize ${bloomBadge}`}
        >
          {question.bloomLevel}
        </span>
        <span
          className={`rounded-full px-2.5 py-0.5 text-xs font-medium capitalize ${diffBadge}`}
        >
          {question.difficulty}
        </span>
        <span className="rounded-full bg-gray-100 px-2.5 py-0.5 text-xs font-medium capitalize text-gray-700">
          {question.questionType.replace("_", " ")}
        </span>
      </div>

      {/* Question text */}
      <p className="mb-5 text-base font-medium text-gray-900 leading-relaxed">
        {question.questionText}
      </p>

      {/* Answer input, varies by question type */}
      <div className="mb-5">
        {question.questionType === "mcq" && question.options && (
          <div className="space-y-2">
            {question.options.map((option, idx) => (
              <label
                key={idx}
                className={`flex cursor-pointer items-center gap-3 rounded-lg border px-4 py-3 text-sm transition-colors ${
                  selected === option.text
                    ? "border-blue-500 bg-blue-50"
                    : "border-gray-200 hover:bg-gray-50"
                }`}
              >
                <input
                  type="radio"
                  name={`question-${question.id}`}
                  value={option.text}
                  checked={selected === option.text}
                  onChange={() => setSelected(option.text)}
                  className="h-4 w-4 border-gray-300 text-blue-600 focus:ring-blue-500"
                />
                <span className="text-gray-800">{option.text}</span>
              </label>
            ))}
          </div>
        )}

        {question.questionType === "true_false" && (
          <div className="flex gap-3">
            {["True", "False"].map((val) => (
              <label
                key={val}
                className={`flex flex-1 cursor-pointer items-center justify-center gap-2 rounded-lg border px-4 py-3 text-sm font-medium transition-colors ${
                  selected === val
                    ? "border-blue-500 bg-blue-50 text-blue-700"
                    : "border-gray-200 text-gray-700 hover:bg-gray-50"
                }`}
              >
                <input
                  type="radio"
                  name={`question-${question.id}`}
                  value={val}
                  checked={selected === val}
                  onChange={() => setSelected(val)}
                  className="h-4 w-4 border-gray-300 text-blue-600 focus:ring-blue-500"
                />
                {val}
              </label>
            ))}
          </div>
        )}

        {question.questionType === "fill_blank" && (
          <input
            type="text"
            value={selected}
            onChange={(e) => setSelected(e.target.value)}
            placeholder="Type your answer..."
            className="w-full rounded-lg border border-gray-300 px-4 py-2.5 text-sm text-gray-900 placeholder-gray-400 outline-none transition-colors focus:border-blue-500 focus:ring-2 focus:ring-blue-500/20"
          />
        )}

        {question.questionType === "short_answer" && (
          <textarea
            rows={3}
            value={selected}
            onChange={(e) => setSelected(e.target.value)}
            placeholder="Write your answer..."
            className="w-full rounded-lg border border-gray-300 px-4 py-2.5 text-sm text-gray-900 placeholder-gray-400 outline-none transition-colors focus:border-blue-500 focus:ring-2 focus:ring-blue-500/20"
          />
        )}

        {question.questionType === "essay" && (
          <textarea
            rows={8}
            value={selected}
            onChange={(e) => setSelected(e.target.value)}
            placeholder="Write your essay response..."
            className="w-full rounded-lg border border-gray-300 px-4 py-2.5 text-sm text-gray-900 placeholder-gray-400 outline-none transition-colors focus:border-blue-500 focus:ring-2 focus:ring-blue-500/20"
          />
        )}

        {/* Fallback for other types (match, assertion_reasoning) */}
        {!["mcq", "true_false", "fill_blank", "short_answer", "essay"].includes(
          question.questionType
        ) && (
          <textarea
            rows={4}
            value={selected}
            onChange={(e) => setSelected(e.target.value)}
            placeholder="Write your answer..."
            className="w-full rounded-lg border border-gray-300 px-4 py-2.5 text-sm text-gray-900 placeholder-gray-400 outline-none transition-colors focus:border-blue-500 focus:ring-2 focus:ring-blue-500/20"
          />
        )}
      </div>

      <button
        type="submit"
        disabled={!selected.trim() || disabled}
        className="w-full rounded-lg bg-blue-600 px-4 py-2.5 text-sm font-medium text-white transition-colors hover:bg-blue-700 disabled:cursor-not-allowed disabled:opacity-50"
      >
        Submit Answer
      </button>
    </form>
  );
}
