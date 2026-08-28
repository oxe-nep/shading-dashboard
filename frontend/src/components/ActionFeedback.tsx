'use client';

import { ActionFeedback as Feedback } from '@/hooks/useWebSocket';

interface ActionFeedbackProps {
  feedback: Feedback | null;
}

export default function ActionFeedbackBanner({ feedback }: ActionFeedbackProps) {
  if (!feedback) return null;

  return (
    <div className={`action-feedback action-feedback-${feedback.type}`}>
      <i
        className={`fas ${
          feedback.type === 'pending'
            ? 'fa-spinner fa-spin'
            : feedback.type === 'success'
              ? 'fa-check-circle'
              : 'fa-exclamation-circle'
        }`}
      />
      <span>{feedback.message}</span>
    </div>
  );
}
