import json
import glob
import pandas as pd
import sys
import os
from datetime import datetime

class Plotter:
  def __init__(self, directory):
    self.PARAMS = ["Jitter", "SocketCount", "RequestedQPS", "Deployed"]
    self.DEFAULT = {"SocketCount": 16, "RequestedQPS": 1000}
    self.PERCENTS = ["50", "75", "90", "99", "99.9"]
    self.data = self.read_data(directory)

  def read_json(self, filename):
    with open(filename) as file:
      result = json.load(file)
    result["Deployed"] = not ("undeployed" in filename)
    return result

  def read_data(self, directory):
    filenames = glob.glob("json/{}/*.json".format(directory))
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

    data = pd.DataFrame(data)

    # clean data with types
    data[self.PERCENTS] = data[self.PERCENTS].astype(float)
    data["RequestedQPS"] = data["RequestedQPS"].astype(int)

    # clean duplicates
    data = data.groupby(self.PARAMS).mean().reset_index()

    # save data as csv
    data.to_csv("csv/{}.csv".format(directory))
    return data

  def select_data(self, jitter, param, default, percent):
    # select data for given jitter and default
    data = self.data
    data = data[data["Jitter"] == jitter]
    data = data[data[default] == self.DEFAULT[default]]

    # set index to given param
    data = data.set_index(param).sort_index()

    # merge deployed and undeployed
    deployed_data = data.loc[data["Deployed"], percent]
    undeployed_data = data.loc[~data["Deployed"], percent]
    data = pd.concat([deployed_data, undeployed_data], axis=1)
    data.columns = ["filter deployed", "filter undeployed"]
    return data

  def plot(self, jitter, param, default, percent):
    # get data
    data = self.select_data(jitter, param, default, percent)

    # make plot
    title = "Latency vs {} with {} = {} and Jitter = {}".format(param, default, self.DEFAULT[default], jitter)
    plot = data.plot(title=title)
    plot.set_ylabel(percent + "th latency (ms)")

    # save figure
    fig = plot.get_figure()
    time = datetime.now().strftime("%m-%d-%Y_%H:%M:%S")
    fig.savefig("figs/jitter={}_param={}_percent={}_time={}.png".format(jitter, param, percent, time))

  def plot_all(self):
    self.plot(True, self.PARAMS[1], self.PARAMS[2], "90")
    self.plot(False, self.PARAMS[1], self.PARAMS[2], "90")
    self.plot(True, self.PARAMS[2], self.PARAMS[1], "90")
    self.plot(False, self.PARAMS[2], self.PARAMS[1], "90")

if __name__ == '__main__':
  if len(sys.argv) > 1:
    directory = sys.argv[1]
  else:
    subdirs = ["json/" + d for d in os.listdir("json")]
    latest = max(subdirs , key=os.path.getmtime)
    directory = latest[5:]
  print("loading directory: " + directory)

  p = Plotter(directory)

  if len(sys.argv) != 6:
    p.plot_all()
  else:
    _, jitter, param, default, percent = sys.argv
    p.plot(jitter, param, default, percent)
