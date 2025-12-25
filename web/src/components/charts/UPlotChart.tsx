import { useEffect, useRef } from 'react';
import uPlot from 'uplot';
import type { AlignedData } from 'uplot';
import 'uplot/dist/uPlot.min.css';

export interface UPlotChartProps {
  data: AlignedData;
  options?: uPlot.Options;
  className?: string;
  onReady?: (chart: uPlot) => void;
}

export function UPlotChart({ data, options, className, onReady }: UPlotChartProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  const chartRef = useRef<uPlot | null>(null);

  useEffect(() => {
    if (!containerRef.current || !data || data.length === 0) return;

    const defaultOptions: uPlot.Options = {
      width: containerRef.current.clientWidth,
      height: 300,
      series: [
        {},
        {
          label: 'Latency (ms)',
          stroke: 'rgb(59, 130, 246)',
          width: 2,
        },
      ],
      axes: [
        {
          grid: { show: true },
        },
        {
          grid: { show: true },
          label: 'Latency (ms)',
        },
      ],
      ...options,
    };

    chartRef.current = new uPlot(defaultOptions, data, containerRef.current);

    if (onReady && chartRef.current) {
      onReady(chartRef.current);
    }

    // Handle resize
    const handleResize = () => {
      if (chartRef.current && containerRef.current) {
        chartRef.current.setSize({
          width: containerRef.current.clientWidth,
          height: 300,
        });
      }
    };

    window.addEventListener('resize', handleResize);

    return () => {
      window.removeEventListener('resize', handleResize);
      if (chartRef.current) {
        chartRef.current.destroy();
        chartRef.current = null;
      }
    };
  }, [data, options, onReady]);

  // Update chart when data changes
  useEffect(() => {
    if (chartRef.current && data) {
      chartRef.current.setData(data);
    }
  }, [data]);

  return <div ref={containerRef} className={className} />;
}
