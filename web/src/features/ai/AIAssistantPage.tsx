import { useState, useRef, useEffect } from 'react';
import { Card, CardContent } from '@/components/ui/Card';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@radix-ui/react-tabs';
import { Loader2, Send, Bot, User, Zap, TrendingUp, AlertTriangle, BarChart } from 'lucide-react';

interface Message {
  id: string;
  role: 'user' | 'assistant';
  content: string;
  timestamp: Date;
}

interface AIResponse {
  session_id: string;
  response: {
    text: string;
    data?: any;
  };
  metadata: {
    query_time_ms: number;
    data_points_analyzed?: number;
    tokens_used?: number;
  };
}

export function AIAssistantPage() {
  const [messages, setMessages] = useState<Message[]>([]);
  const [input, setInput] = useState('');
  const [isLoading, setIsLoading] = useState(false);
  const [sessionId, setSessionId] = useState<string>('');
  const messagesEndRef = useRef<HTMLDivElement>(null);

  const scrollToBottom = () => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  };

  useEffect(() => {
    scrollToBottom();
  }, [messages]);

  const sendMessage = async () => {
    if (!input.trim() || isLoading) return;

    const userMessage: Message = {
      id: Date.now().toString(),
      role: 'user',
      content: input,
      timestamp: new Date(),
    };

    setMessages(prev => [...prev, userMessage]);
    setInput('');
    setIsLoading(true);

    try {
      const response = await fetch('/api/v1/ai/query', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': 'Bearer demo-key',
        },
        body: JSON.stringify({
          query: input,
          session_id: sessionId,
          time_range: {
            start: new Date(Date.now() - 24 * 60 * 60 * 1000).toISOString(),
            end: new Date().toISOString(),
          },
        }),
      });

      if (!response.ok) {
        throw new Error('Failed to get AI response');
      }

      const aiResponse: AIResponse = await response.json();
      
      if (!sessionId) {
        setSessionId(aiResponse.session_id);
      }

      const assistantMessage: Message = {
        id: (Date.now() + 1).toString(),
        role: 'assistant',
        content: aiResponse.response.text,
        timestamp: new Date(),
      };

      setMessages(prev => [...prev, assistantMessage]);
    } catch (error) {
      const errorMessage: Message = {
        id: (Date.now() + 1).toString(),
        role: 'assistant',
        content: 'Sorry, I encountered an error while processing your request. Please try again.',
        timestamp: new Date(),
      };
      setMessages(prev => [...prev, errorMessage]);
    } finally {
      setIsLoading(false);
    }
  };

  const handleKeyPress = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      sendMessage();
    }
  };

  const exampleQueries = [
    {
      icon: TrendingUp,
      title: "Performance Trends",
      query: "Show me performance trends for the past week"
    },
    {
      icon: AlertTriangle,
      title: "Worst Performers",
      query: "Which clients had the worst performance today?"
    },
    {
      icon: BarChart,
      title: "Client Comparison",
      query: "Compare client performance across all targets"
    },
    {
      icon: Zap,
      title: "Anomaly Detection",
      query: "Are there any performance anomalies in the last hour?"
    }
  ];

  const startExampleQuery = (query: string) => {
    setInput(query);
  };

  const clearChat = () => {
    setMessages([]);
    setSessionId('');
  };

  return (
    <div className="h-full flex flex-col">
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-3xl font-bold">AI Assistant</h1>
          <p className="text-muted-foreground mt-1">
            Analyze network telemetry data with natural language queries
          </p>
        </div>
        {messages.length > 0 && (
          <button
            onClick={clearChat}
            className="px-4 py-2 text-sm bg-gray-100 hover:bg-gray-200 rounded-md"
          >
            Clear Chat
          </button>
        )}
      </div>

      <div className="flex-1 flex gap-6">
        {/* Chat Interface */}
        <div className="flex-1 flex flex-col">
          <Card className="flex-1 flex flex-col">
            <CardContent className="flex-1 flex flex-col p-4">
              {/* Messages */}
              <div className="flex-1 overflow-auto mb-4 space-y-4">
                {messages.length === 0 && (
                  <div className="h-full flex items-center justify-center text-center">
                    <div className="max-w-md">
                      <Bot className="mx-auto h-12 w-12 text-blue-500 mb-4" />
                      <h3 className="text-lg font-medium mb-2">Welcome to the AI Assistant</h3>
                      <p className="text-muted-foreground mb-4">
                        Ask me anything about your network telemetry data. I can analyze performance, 
                        identify issues, and help you understand trends.
                      </p>
                      <p className="text-sm text-muted-foreground">
                        Try one of the example queries on the right to get started!
                      </p>
                    </div>
                  </div>
                )}

                {messages.map((message) => (
                  <div
                    key={message.id}
                    className={`flex gap-3 ${
                      message.role === 'user' ? 'justify-end' : 'justify-start'
                    }`}
                  >
                    {message.role === 'assistant' && (
                      <div className="w-8 h-8 rounded-full bg-blue-500 flex items-center justify-center flex-shrink-0">
                        <Bot className="w-4 h-4 text-white" />
                      </div>
                    )}
                    <div
                      className={`max-w-[80%] rounded-lg p-3 ${
                        message.role === 'user'
                          ? 'bg-blue-500 text-white ml-auto'
                          : 'bg-gray-100 text-gray-900'
                      }`}
                    >
                      <div className="whitespace-pre-wrap">{message.content}</div>
                      <div className={`text-xs mt-1 ${
                        message.role === 'user' ? 'text-blue-200' : 'text-gray-500'
                      }`}>
                        {message.timestamp.toLocaleTimeString()}
                      </div>
                    </div>
                    {message.role === 'user' && (
                      <div className="w-8 h-8 rounded-full bg-gray-500 flex items-center justify-center flex-shrink-0">
                        <User className="w-4 h-4 text-white" />
                      </div>
                    )}
                  </div>
                ))}

                {isLoading && (
                  <div className="flex gap-3 justify-start">
                    <div className="w-8 h-8 rounded-full bg-blue-500 flex items-center justify-center flex-shrink-0">
                      <Bot className="w-4 h-4 text-white" />
                    </div>
                    <div className="bg-gray-100 rounded-lg p-3">
                      <div className="flex items-center gap-2">
                        <Loader2 className="w-4 h-4 animate-spin" />
                        <span className="text-sm text-gray-600">Analyzing data...</span>
                      </div>
                    </div>
                  </div>
                )}
                <div ref={messagesEndRef} />
              </div>

              {/* Input */}
              <div className="flex gap-2">
                <textarea
                  value={input}
                  onChange={(e) => setInput(e.target.value)}
                  onKeyPress={handleKeyPress}
                  placeholder="Ask about network performance, client issues, trends..."
                  className="flex-1 p-3 border rounded-lg resize-none focus:outline-none focus:ring-2 focus:ring-blue-500"
                  rows={2}
                  disabled={isLoading}
                />
                <button
                  onClick={sendMessage}
                  disabled={!input.trim() || isLoading}
                  className="px-4 py-2 bg-blue-500 text-white rounded-lg hover:bg-blue-600 disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-2"
                >
                  <Send className="w-4 h-4" />
                  Send
                </button>
              </div>
            </CardContent>
          </Card>
        </div>

        {/* Sidebar with Example Queries */}
        <div className="w-80">
          <Tabs defaultValue="examples" className="w-full">
            <TabsList className="grid w-full grid-cols-1">
              <TabsTrigger value="examples">Example Queries</TabsTrigger>
            </TabsList>
            <TabsContent value="examples">
              <Card>
                <CardContent className="p-4">
                  <h3 className="font-medium mb-3">Try these queries:</h3>
                  <div className="space-y-2">
                    {exampleQueries.map((example, index) => {
                      const Icon = example.icon;
                      return (
                        <button
                          key={index}
                          onClick={() => startExampleQuery(example.query)}
                          className="w-full text-left p-3 rounded-lg border hover:bg-gray-50 transition-colors"
                        >
                          <div className="flex items-start gap-3">
                            <Icon className="w-5 h-5 text-blue-500 mt-0.5 flex-shrink-0" />
                            <div>
                              <div className="font-medium text-sm">{example.title}</div>
                              <div className="text-xs text-muted-foreground mt-1">
                                {example.query}
                              </div>
                            </div>
                          </div>
                        </button>
                      );
                    })}
                  </div>

                  <div className="mt-6 p-3 bg-blue-50 rounded-lg">
                    <h4 className="font-medium text-sm text-blue-900 mb-2">💡 Tips</h4>
                    <ul className="text-xs text-blue-700 space-y-1">
                      <li>• Ask about specific time ranges</li>
                      <li>• Compare different clients or targets</li>
                      <li>• Request trend analysis</li>
                      <li>• Inquire about anomalies or issues</li>
                    </ul>
                  </div>
                </CardContent>
              </Card>
            </TabsContent>
          </Tabs>
        </div>
      </div>
    </div>
  );
}