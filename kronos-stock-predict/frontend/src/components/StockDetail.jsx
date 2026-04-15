import React, { useState, useEffect } from 'react'

function StockDetail({ stock, onBack }) {
  const [klines, setKlines] = useState([])
  const [predictions, setPredictions] = useState(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    fetchData()
  }, [stock.code])

  const fetchData = async () => {
    setLoading(true)
    try {
      const [klineRes, predRes] = await Promise.all([
        fetch(`/api/kline/${stock.code}`),
        fetch(`/api/prediction/${stock.code}`)
      ])

      if (klineRes.ok) {
        const kdata = await klineRes.json()
        setKlines(kdata)
      }

      if (predRes.ok) {
        const pdata = await predRes.json()
        setPredictions(pdata)
      }
    } catch (err) {
      console.error('Failed to fetch data:', err)
    } finally {
      setLoading(false)
    }
  }

  const getDirectionIcon = (direction) => {
    if (direction === 'up') return '↑'
    if (direction === 'down') return '↓'
    return '→'
  }

  const getDirectionColor = (direction) => {
    if (direction === 'up') return 'text-red-400'
    if (direction === 'down') return 'text-green-400'
    return 'text-gray-400'
  }

  const getScoreColor = (score) => {
    if (score >= 0.7) return 'text-green-400'
    if (score >= 0.5) return 'text-yellow-400'
    return 'text-red-400'
  }

  return (
    <div>
      <div className="bg-slate-800 rounded-lg p-6 mb-6">
        <div className="flex items-start justify-between">
          <div>
            <h2 className="text-2xl font-bold text-white">{stock.name}</h2>
            <p className="text-gray-400 font-mono mt-1">{stock.code}</p>
          </div>
          <div className="text-right">
            <p className="text-3xl font-bold text-white">
              {stock.price > 0 ? stock.price.toFixed(2) : '-'}
            </p>
            <p className={`text-lg ${stock.change_pct > 0 ? 'text-red-400' : stock.change_pct < 0 ? 'text-green-400' : 'text-gray-400'}`}>
              {stock.change_pct !== 0 ? `${stock.change_pct > 0 ? '+' : ''}${stock.change_pct.toFixed(2)}%` : '-'}
            </p>
          </div>
        </div>
      </div>

      {loading && (
        <div className="text-center py-8">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-500 mx-auto"></div>
          <p className="mt-4 text-gray-400">加载中...</p>
        </div>
      )}

      {predictions && predictions.predictions && predictions.predictions.length > 0 && (
        <div className="grid grid-cols-1 md:grid-cols-2 gap-6 mb-6">
          {predictions.predictions.map((pred) => (
            <div key={pred.lookback} className="bg-slate-800 rounded-lg p-6">
              <div className="flex items-center justify-between mb-4">
                <h3 className="text-lg font-medium text-white">
                  近 {pred.lookback} 根K线预测
                </h3>
                <div className={`text-2xl ${getDirectionColor(pred.direction)}`}>
                  {getDirectionIcon(pred.direction)}
                </div>
              </div>

              <div className="space-y-3">
                <div className="grid grid-cols-4 gap-2 text-sm">
                  <div>
                    <p className="text-gray-500">开盘</p>
                    <p className="text-white font-mono">{pred.next_open.toFixed(2)}</p>
                  </div>
                  <div>
                    <p className="text-gray-500">最高</p>
                    <p className="text-white font-mono">{pred.next_high.toFixed(2)}</p>
                  </div>
                  <div>
                    <p className="text-gray-500">最低</p>
                    <p className="text-white font-mono">{pred.next_low.toFixed(2)}</p>
                  </div>
                  <div>
                    <p className="text-gray-500">收盘</p>
                    <p className="text-white font-mono">{pred.next_close.toFixed(2)}</p>
                  </div>
                </div>

                <div className="border-t border-slate-700 pt-3">
                  <div className="flex justify-between items-center">
                    <span className="text-gray-400">预测涨跌幅</span>
                    <span className={`text-xl font-bold ${pred.change_pct > 0 ? 'text-red-400' : pred.change_pct < 0 ? 'text-green-400' : 'text-gray-400'}`}>
                      {pred.change_pct > 0 ? '+' : ''}{pred.change_pct.toFixed(2)}%
                    </span>
                  </div>
                </div>

                <div className="border-t border-slate-700 pt-3">
                  <div className="flex justify-between items-center">
                    <span className="text-gray-400">预测分数</span>
                    <span className={`text-xl font-bold ${getScoreColor(pred.score)}`}>
                      {(pred.score * 100).toFixed(0)}%
                    </span>
                  </div>
                  <div className="mt-2 bg-slate-700 rounded-full h-2 overflow-hidden">
                    <div
                      className={`h-full transition-all ${
                        pred.score >= 0.7 ? 'bg-green-500' : pred.score >= 0.5 ? 'bg-yellow-500' : 'bg-red-500'
                      }`}
                      style={{ width: `${pred.score * 100}%` }}
                    ></div>
                  </div>
                </div>

                {pred.predicted_at && (
                  <div className="text-xs text-gray-500 mt-2">
                    预测时间: {new Date(pred.predicted_at).toLocaleString('zh-CN')}
                  </div>
                )}
              </div>
            </div>
          ))}
        </div>
      )}

      {predictions && (!predictions.predictions || predictions.predictions.length === 0) && (
        <div className="bg-slate-800 rounded-lg p-6 mb-6 text-center">
          <p className="text-gray-400">暂无预测数据</p>
        </div>
      )}

      {klines.length > 0 && (
        <div className="bg-slate-800 rounded-lg p-6">
          <h3 className="text-lg font-medium text-white mb-4">最近K线数据</h3>
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead className="bg-slate-700">
                <tr>
                  <th className="px-3 py-2 text-left text-gray-300">日期</th>
                  <th className="px-3 py-2 text-right text-gray-300">开盘</th>
                  <th className="px-3 py-2 text-right text-gray-300">最高</th>
                  <th className="px-3 py-2 text-right text-gray-300">最低</th>
                  <th className="px-3 py-2 text-right text-gray-300">收盘</th>
                  <th className="px-3 py-2 text-right text-gray-300">成交量</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-700">
                {klines.slice(-10).reverse().map((k, i) => (
                  <tr key={i}>
                    <td className="px-3 py-2 text-gray-400">{k.timestamp}</td>
                    <td className="px-3 py-2 text-right text-white font-mono">{k.open.toFixed(2)}</td>
                    <td className="px-3 py-2 text-right text-white font-mono">{k.high.toFixed(2)}</td>
                    <td className="px-3 py-2 text-right text-white font-mono">{k.low.toFixed(2)}</td>
                    <td className="px-3 py-2 text-right text-white font-mono">{k.close.toFixed(2)}</td>
                    <td className="px-3 py-2 text-right text-gray-400 font-mono">{(k.volume/10000).toFixed(0)}万</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}
    </div>
  )
}

export default StockDetail
