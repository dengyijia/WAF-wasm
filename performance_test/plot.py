import json
import glob
import pandas as pd

class Plotter:
  def __init__(self):
    self.PARAMS = ["Jitter", "SocketCount", "RequestedQPS"]
    self.DEFAULT = {"SocketCount": 16, "RequestedQPS": 1000}
    self.PERCENTS = ["50", "75", "90", "99", "99.9"]
    self.data = self.read_data()

  def read_json(self, filename):
    with open(filename) as file:
      result = json.load(file)
    return result

  def read_data(self):
    filenames = glob.glob("*.json")
    data = {param: [] for param in self.PARAMS + self.PERCENTS}
    for filename in filenames:
      result = self.read_json(filename)
      # record the parameters
      for param in self.PARAMS:
        data[param].append(result[param])
      # record the percentiles
      percentiles = result["DurationHistogram"]["Percentiles"]
      for percentile in percentiles:
        data[str(percentile["Percentile"])].append(percentile["Value"])
    return pd.DataFrame(data)

  def plot(self, jitter, param, default):
    data = self.data
    data = data[data["Jitter"] == jitter]
    data = data[data[default] == self.DEFAULT[default]]
    data = data.set_index(param)
    plot = data.plot.line()
    fig = plot.get_figure()
    fig.savefig("jitter={}_param={}.png".format(jitter, param))

  def plot_all(self):
    self.plot(True, self.PARAMS[1], self.PARAMS[2])
    self.plot(False, self.PARAMS[1], self.PARAMS[2])
    self.plot(True, self.PARAMS[2], self.PARAMS[1])
    self.plot(False, self.PARAMS[2], self.PARAMS[1])
