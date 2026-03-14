"use client";

import { useState, useEffect, use } from "react";
import {
  getBook,
  getRandomQuestions,
  submitAnswer,
} from "@/lib/api";
import type { Book, Question, UserAnswer } from "@/lib/types";
import QuestionCard from "@/components/QuestionCard";
import ScoreBreakdown from "@/components/ScoreBreakdown";

export default function PracticePage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = use(params);
  const [book, setBook] = useState<Book | null>(null);
  const [questions, setQuestions] = useState<Question[]>([]);
  const [currentIndex, setCurrentIndex] = useState(0);
  const [result, setResult] = useState<UserAnswer | null>(null);
  const [loading, setLoading] = useState(true);
  const [submitting, setSubmitting] = useState(false);
  const [startTime, setStartTime] = useState<number>(Date.now());

  useEffect(() => {
    async function load() {
      try {
        const [bookData, questionData] = await Promise.all([
          getBook(id),
          getRandomQuestions({ bookId: id, count: 10 }),
        ]);
        setBook(bookData);
        setQuestions(questionData.data || []);
      } catch {
        // Error
      } finally {
        setLoading(false);
      }
    }
    load();
  }, [id]);

  useEffect(() => {
    setStartTime(Date.now());
    setResult(null);
  }, [currentIndex]);

  async function handleAnswer(answer: string) {
    if (!questions[currentIndex]) return;
    setSubmitting(true);

    try {
      const timeTaken = Math.floor((Date.now() - startTime) / 1000);
      const res = await submitAnswer({
        questionId: questions[currentIndex].id,
        answerText: answer,
        timeTaken,
      });
      setResult(res);
    } catch (err) {
      alert(err instanceof Error ? err.message : "Failed to submit answer");
    } finally {
      setSubmitting(false);
    }
  }

  function nextQuestion() {
    if (currentIndex < questions.length - 1) {
      setCurrentIndex(currentIndex + 1);
    }
  }

  if (loading) return <div className="text-center py-12">Loading...</div>;
  if (!book) return <div className="text-center py-12">Book not found</div>;
  if (questions.length === 0)
    return (
      <div className="text-center py-12">
        No questions available for this book
      </div>
    );

  const question = questions[currentIndex];

  return (
    <div className="max-w-3xl mx-auto">
      <div className="mb-6">
        <h1 className="text-xl font-bold">{book.title}</h1>
        <p className="text-sm text-gray-500">
          Question {currentIndex + 1} of {questions.length}
        </p>
        <div className="w-full bg-gray-200 rounded-full h-2 mt-2">
          <div
            className="bg-blue-600 h-2 rounded-full transition-all"
            style={{
              width: `${((currentIndex + 1) / questions.length) * 100}%`,
            }}
          />
        </div>
      </div>

      <div className="mb-4 flex gap-2">
        <span className="text-xs bg-purple-100 text-purple-700 px-2 py-0.5 rounded">
          {question.bloomLevel}
        </span>
        <span className="text-xs bg-orange-100 text-orange-700 px-2 py-0.5 rounded">
          {question.difficulty}
        </span>
        <span className="text-xs bg-gray-100 text-gray-700 px-2 py-0.5 rounded">
          {question.questionType.replace("_", " ")}
        </span>
      </div>

      {!result ? (
        <QuestionCard
          question={question}
          onAnswer={handleAnswer}
          disabled={submitting}
        />
      ) : (
        <div className="space-y-4">
          <ScoreBreakdown answer={result} />

          {question.explanation && (
            <div className="bg-blue-50 rounded-lg p-4">
              <h3 className="font-medium text-blue-800 mb-1">Explanation</h3>
              <p className="text-sm text-blue-700">{question.explanation}</p>
            </div>
          )}

          {currentIndex < questions.length - 1 ? (
            <button
              onClick={nextQuestion}
              className="w-full bg-blue-600 text-white py-3 rounded-lg hover:bg-blue-700"
            >
              Next Question
            </button>
          ) : (
            <div className="text-center py-4">
              <p className="text-lg font-semibold mb-2">Practice Complete!</p>
              <a
                href="/dashboard"
                className="text-blue-600 hover:underline"
              >
                View your stats
              </a>
            </div>
          )}
        </div>
      )}
    </div>
  );
}
