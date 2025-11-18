import React, { useEffect, useRef, useState, useMemo } from 'react';
import { Play, Pause, ChevronLeft, ChevronRight, Settings, Info, MousePointer2 } from 'lucide-react';

// --- Constants & Types ---

const SCALE_FACTOR = 100.0;

interface Frame {
  width: number;
  height: number;
  data: Float32Array; // Interleaved x, y
}

// --- decoder.ts logic (Embedded) ---

/**
 * Decodes your custom 16-bit packed PNG format.
 * R=X_high, G=X_low, B=Y_high, A=Y_low
 * Note: In a real browser environment fetching from a server, 
 * createImageBitmap with { premultiplyAlpha: 'none' } is essential.
 */
async function unmarshalVectorFrame(blob: Blob): Promise<Frame> {
  const bitmap = await createImageBitmap(blob, {
    premultiplyAlpha: 'none',
    colorSpaceConversion: 'none',
    resizeQuality: 'pixelated',
  });

  const width = bitmap.width;
  const height = bitmap.height;

  // Use an offscreen canvas to extract raw bytes
  const canvas = document.createElement('canvas');
  canvas.width = width;
  canvas.height = height;
  const ctx = canvas.getContext('2d', { willReadFrequently: true });
  if (!ctx) throw new Error("No Canvas Context");

  ctx.drawImage(bitmap, 0, 0);
  const imgData = ctx.getImageData(0, 0, width, height);
  const pixels = imgData.data;

  const vectorData = new Float32Array(width * height * 2);

  for (let i = 0; i < width * height; i++) {
    const pIdx = i * 4;
    const vIdx = i * 2;

    // Extract bytes
    const r = pixels[pIdx];     // X High
    const g = pixels[pIdx + 1]; // X Low
    const b = pixels[pIdx + 2]; // Y High
    const a = pixels[pIdx + 3]; // Y Low

    // Reconstruct uint16
    let xUint = (r << 8) | g;
    let yUint = (b << 8) | a;

    // Handle Two's Complement for int16 range (-32768 to 32767)
    const xInt = xUint > 32767 ? xUint - 65536 : xUint;
    const yInt = yUint > 32767 ? yUint - 65536 : yUint;

    vectorData[vIdx] = xInt / SCALE_FACTOR;
    vectorData[vIdx + 1] = yInt / SCALE_FACTOR;
  }

  return { width, height, data: vectorData };
}

// --- Mock Data Generator (For Demo Only) ---

function generateMockFrame(t: number, width: number, height: number): Frame {
  const data = new Float32Array(width * height * 2);

  // Create a swirling vector field that evolves over time t
  const timeOffset = t * 0.5;
  const centerX = width / 2;
  const centerY = height / 2;

  for (let y = 0; y < height; y++) {
    for (let x = 0; x < width; x++) {
      const idx = (y * width + x) * 2;

      // Normalized coordinates -1 to 1
      const nx = (x - centerX) / centerX;
      const ny = (y - centerY) / centerY;

      // Simple vortex math + sine wave ripple over time
      const dist = Math.sqrt(nx * nx + ny * ny);
      const angle = Math.atan2(ny, nx);

      // Velocity components
      const speed = 20.0 * Math.exp(-dist * 2) * Math.cos(dist * 10 - timeOffset);

      // Tangential flow (vortex) + Radial flow (expansion)
      const vx = -Math.sin(angle) * speed + Math.cos(angle) * (Math.sin(timeOffset) * 5);
      const vy = Math.cos(angle) * speed + Math.sin(angle) * (Math.sin(timeOffset) * 5);

      data[idx] = vx;
      data[idx + 1] = vy;
    }
  }

  return { width, height, data };
}

// --- Components ---

const VectorCanvas = ({
  frame,
  stride,
  scale,
  onHover
}: {
  frame: Frame | null,
  stride: number,
  scale: number,
  onHover: (x: number, y: number, vx: number, vy: number) => void
}) => {
  const canvasRef = useRef<HTMLCanvasElement>(null);

  // Handle Mouse Hover
  const handleMouseMove = (e: React.MouseEvent) => {
    if (!frame || !canvasRef.current) return;
    const rect = canvasRef.current.getBoundingClientRect();
    const x = Math.floor((e.clientX - rect.left) * (frame.width / rect.width));
    const y = Math.floor((e.clientY - rect.top) * (frame.height / rect.height));

    if (x >= 0 && x < frame.width && y >= 0 && y < frame.height) {
      const idx = (y * frame.width + x) * 2;
      onHover(x, y, frame.data[idx], frame.data[idx + 1]);
    }
  };

  useEffect(() => {
    if (!frame || !canvasRef.current) return;
    const ctx = canvasRef.current.getContext('2d');
    if (!ctx) return;

    // Clear
    ctx.fillStyle = '#111827'; // Tailwind gray-900
    ctx.fillRect(0, 0, frame.width, frame.height);

    // Draw
    for (let y = 0; y < frame.height; y += stride) {
      for (let x = 0; x < frame.width; x += stride) {
        const idx = (y * frame.width + x) * 2;
        const vx = frame.data[idx];
        const vy = frame.data[idx + 1];

        const mag = Math.sqrt(vx * vx + vy * vy);
        if (mag < 0.1) continue; // Skip tiny vectors

        // Color Mapping
        const angle = Math.atan2(vy, vx);
        const degrees = (angle * 180 / Math.PI + 360) % 360;

        ctx.strokeStyle = `hsl(${degrees}, 80%, 60%)`;
        ctx.fillStyle = `hsl(${degrees}, 80%, 60%)`;
        ctx.lineWidth = 1;

        // Draw Arrow
        const len = Math.min(stride * 1.2, mag * scale);
        const cx = x + stride / 2; // Center in the grid cell
        const cy = y + stride / 2;

        const endX = cx + Math.cos(angle) * len;
        const endY = cy + Math.sin(angle) * len;

        ctx.beginPath();
        ctx.moveTo(cx, cy);
        ctx.lineTo(endX, endY);
        ctx.stroke();

        // Arrowhead
        const headLen = Math.max(2, len * 0.25);
        ctx.beginPath();
        ctx.moveTo(endX, endY);
        ctx.lineTo(
          endX - headLen * Math.cos(angle - Math.PI / 6),
          endY - headLen * Math.sin(angle - Math.PI / 6)
        );
        ctx.lineTo(
          endX - headLen * Math.cos(angle + Math.PI / 6),
          endY - headLen * Math.sin(angle + Math.PI / 6)
        );
        ctx.fill();
      }
    }
  }, [frame, stride, scale]);

  if (!frame) return <div className="flex items-center justify-center h-64 text-gray-500">No Data</div>;

  return (
    <canvas
      ref={canvasRef}
      width={frame.width}
      height={frame.height}
      className="w-full h-auto border border-gray-700 rounded shadow-lg cursor-crosshair"
      style={{ imageRendering: 'pixelated' }}
      onMouseMove={handleMouseMove}
      onMouseLeave={() => onHover(-1, -1, 0, 0)}
    />
  );
};

export default function App() {
  // --- State ---
  const [timeStep, setTimeStep] = useState(0);
  const [maxTime] = useState(20);
  const [isPlaying, setIsPlaying] = useState(false);
  const [playbackSpeed, setPlaybackSpeed] = useState(200); // ms per frame

  const [stride, setStride] = useState(16);
  const [arrowScale, setArrowScale] = useState(1.5);

  const [currentFrame, setCurrentFrame] = useState<Frame | null>(null);
  const [hoverInfo, setHoverInfo] = useState<{ x: number, y: number, vx: number, vy: number } | null>(null);

  // --- Data Loading Logic ---

  useEffect(() => {
    let isMounted = true;

    const loadFrame = async () => {
      // ---------------------------------------------------------
      // SWITCH HERE: To use your real Go server, uncomment below:
      // ---------------------------------------------------------
      /*
      try {
        const response = await fetch(`http://localhost:8080/vectors?t=${timeStep}`);
        const blob = await response.blob();
        const frame = await unmarshalVectorFrame(blob);
        if (isMounted) setCurrentFrame(frame);
      } catch (e) {
        console.error("Failed to load frame", e);
      }
      */

      // --- DEMO MODE: Generating math data locally ---
      const mock = generateMockFrame(timeStep, 256, 256);
      if (isMounted) setCurrentFrame(mock);
    };

    loadFrame();

    return () => { isMounted = false; };
  }, [timeStep]);

  // --- Playback Loop ---
  useEffect(() => {
    let interval: number;
    if (isPlaying) {
      interval = setInterval(() => {
        setTimeStep(prev => (prev + 1) % maxTime);
      }, playbackSpeed);
    }
    return () => clearInterval(interval);
  }, [isPlaying, maxTime, playbackSpeed]);

  // --- Helpers ---
  const togglePlay = () => setIsPlaying(!isPlaying);
  const nextStep = () => { setIsPlaying(false); setTimeStep(t => (t + 1) % maxTime); };
  const prevStep = () => { setIsPlaying(false); setTimeStep(t => (t - 1 < 0 ? maxTime - 1 : t - 1)); };

  return (
    <div className="min-h-screen bg-gray-950 text-gray-100 p-6 font-sans">
      <div className="max-w-5xl mx-auto">

        {/* Header */}
        <header className="mb-6 flex justify-between items-end">
          <div>
            <h1 className="text-2xl font-bold text-blue-400 mb-1">Vector Sequence Viewer</h1>
            <p className="text-sm text-gray-400">Visualizing x,y vector fields over time (t)</p>
          </div>
          <div className="text-right text-xs text-gray-500">
            <div className="flex items-center gap-2 justify-end mb-1">
              <div className="w-3 h-3 rounded-full bg-red-500"></div> +X
              <div className="w-3 h-3 rounded-full bg-green-500"></div> +Y
              <div className="w-3 h-3 rounded-full bg-cyan-500"></div> -X
              <div className="w-3 h-3 rounded-full bg-purple-500"></div> -Y
            </div>
            Resolution: {currentFrame ? `${currentFrame.width}x${currentFrame.height}` : 'Loading...'}
          </div>
        </header>

        {/* Main Layout */}
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">

          {/* Left: Controls */}
          <div className="lg:col-span-1 space-y-6">

            {/* Playback Controls */}
            <div className="bg-gray-900 p-5 rounded-lg border border-gray-800">
              <div className="flex justify-between items-center mb-4">
                <span className="text-sm font-medium text-gray-300">Time Control</span>
                <span className="text-xl font-mono text-blue-400">t = {timeStep}</span>
              </div>

              <div className="flex items-center gap-2 mb-4">
                <button onClick={prevStep} className="p-2 hover:bg-gray-800 rounded transition"><ChevronLeft size={20} /></button>
                <button
                  onClick={togglePlay}
                  className={`flex-1 flex items-center justify-center gap-2 p-2 rounded font-medium transition ${isPlaying ? 'bg-red-900/50 text-red-200 hover:bg-red-900/70' : 'bg-blue-600 hover:bg-blue-500 text-white'}`}
                >
                  {isPlaying ? <><Pause size={18} /> Pause</> : <><Play size={18} /> Play</>}
                </button>
                <button onClick={nextStep} className="p-2 hover:bg-gray-800 rounded transition"><ChevronRight size={20} /></button>
              </div>

              <input
                type="range"
                min="0" max={maxTime - 1}
                value={timeStep}
                onChange={(e) => { setIsPlaying(false); setTimeStep(parseInt(e.target.value)); }}
                className="w-full h-2 bg-gray-700 rounded-lg appearance-none cursor-pointer accent-blue-500"
              />
              <div className="flex justify-between text-xs text-gray-500 mt-2">
                <span>0</span>
                <span>{maxTime}</span>
              </div>
            </div>

            {/* Display Settings */}
            <div className="bg-gray-900 p-5 rounded-lg border border-gray-800">
              <div className="flex items-center gap-2 mb-4 text-sm font-medium text-gray-300">
                <Settings size={16} /> Visualization Settings
              </div>

              <div className="space-y-4">
                <div>
                  <label className="block text-xs text-gray-400 mb-1">Stride (Density): {stride}px</label>
                  <input
                    type="range" min="4" max="64" step="4"
                    value={stride}
                    onChange={(e) => setStride(parseInt(e.target.value))}
                    className="w-full h-1 bg-gray-700 rounded appearance-none accent-gray-400"
                  />
                </div>
                <div>
                  <label className="block text-xs text-gray-400 mb-1">Arrow Scale: {arrowScale.toFixed(1)}x</label>
                  <input
                    type="range" min="0.1" max="5.0" step="0.1"
                    value={arrowScale}
                    onChange={(e) => setArrowScale(parseFloat(e.target.value))}
                    className="w-full h-1 bg-gray-700 rounded appearance-none accent-gray-400"
                  />
                </div>
                <div>
                  <label className="block text-xs text-gray-400 mb-1">Playback Speed: {playbackSpeed}ms</label>
                  <input
                    type="range" min="50" max="1000" step="50"
                    value={playbackSpeed}
                    onChange={(e) => setPlaybackSpeed(parseInt(e.target.value))}
                    className="w-full h-1 bg-gray-700 rounded appearance-none accent-gray-400"
                  />
                </div>
              </div>
            </div>

            {/* Inspector */}
            <div className="bg-gray-900 p-5 rounded-lg border border-gray-800 min-h-[120px]">
              <div className="flex items-center gap-2 mb-2 text-sm font-medium text-gray-300">
                <MousePointer2 size={16} /> Inspector
              </div>
              {hoverInfo && hoverInfo.x !== -1 ? (
                <div className="space-y-1 font-mono text-sm">
                  <div className="flex justify-between">
                    <span className="text-gray-500">Pos:</span>
                    <span>({hoverInfo.x}, {hoverInfo.y})</span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-gray-500">Vec X:</span>
                    <span className={hoverInfo.vx < 0 ? 'text-red-400' : 'text-green-400'}>{hoverInfo.vx.toFixed(2)}</span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-gray-500">Vec Y:</span>
                    <span className={hoverInfo.vy < 0 ? 'text-red-400' : 'text-green-400'}>{hoverInfo.vy.toFixed(2)}</span>
                  </div>
                  <div className="flex justify-between border-t border-gray-700 mt-2 pt-2">
                    <span className="text-gray-500">Mag:</span>
                    <span className="text-white">{Math.sqrt(hoverInfo.vx ** 2 + hoverInfo.vy ** 2).toFixed(2)}</span>
                  </div>
                </div>
              ) : (
                <div className="text-xs text-gray-600 italic text-center mt-4">
                  Hover over the grid to inspect vector values
                </div>
              )}
            </div>

          </div>

          {/* Right: Canvas */}
          <div className="lg:col-span-2 bg-black rounded-lg overflow-hidden border border-gray-800 shadow-2xl relative">
            <VectorCanvas
              frame={currentFrame}
              stride={stride}
              scale={arrowScale}
              onHover={(x, y, vx, vy) => setHoverInfo({ x, y, vx, vy })}
            />
            <div className="absolute top-2 right-2 px-2 py-1 bg-black/60 text-gray-400 text-xs rounded backdrop-blur-sm pointer-events-none">
              Canvas Render
            </div>
          </div>

        </div>
      </div>
    </div>
  );
}
